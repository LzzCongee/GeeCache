package geecache

import (
	"fmt"
	"geecache/consistenthash"
	pb "geecache/geecachepb"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"crypto/tls"
	"crypto/x509"

	"github.com/golang/protobuf/proto"
)

const (
	defaultBasePath = "/_geecache/" // 默认路径
	defaultReplicas = 50
)

// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:8000"
	self        string
	basePath    string
	mu          sync.Mutex // guards peers and httpGetters
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.2:8008"

	client      *http.Client    // 支持自定义 TLS Client
}

// NewHTTPPool initializes an HTTP pool of peers.
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,            // 本机地址
		basePath: defaultBasePath, // 默认路径
	}
}

func NewHTTPPoolWithTLS(self, caFile string) *HTTPPool {
    client := newTLSClient(caFile)
    return &HTTPPool{self: self, basePath: defaultBasePath, client: client}
}

func newTLSClient(caFile string) *http.Client {
    pool := x509.NewCertPool()
    pem, err := ioutil.ReadFile(caFile)
    pool.AppendCertsFromPEM(pem)
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{
            RootCAs:    pool,
            MinVersion: tls.VersionTLS12,
        },
    }
    return &http.Client{Transport: tr}
}



// Log info with server name
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http requests
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path) // 打印请求方法和路径
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName) // 获取指定组
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// 处理不同的HTTP方法
	switch r.Method {
	case http.MethodGet:
		// 处理GET请求，获取缓存数据
		view, err := group.Get(key) // 从指定组中获取指定值
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Write the value to the response body as a proto message.
		body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(body)

	case http.MethodPut:
		// 处理PUT请求，存储热点数据
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("reading request body: %v", err), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// 解析请求体中的protobuf数据
		res := &pb.Response{}
		if err = proto.Unmarshal(body, res); err != nil {
			http.Error(w, fmt.Sprintf("decoding request body: %v", err), http.StatusBadRequest)
			return
		}

		// 将数据添加到本地缓存
		value := ByteView{b: cloneBytes(res.Value)}
		group.mainCache.add(key, value)

		p.Log("Stored hot spot data for group=%s, key=%s", groupName, key)
		w.WriteHeader(http.StatusOK)

	default:
		w.Header().Set("Allow", "GET, PUT")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Set updates the pool's list of peers.
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil) // 创建一个一致性哈希
	p.peers.Add(peers...)                              // 添加节点到一致性哈希
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		// 为每个节点创建一个 HTTP 客户端，地址:http://10.0.0.2:8008/_geecache/
		// p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath, client: p.client}
	}
}

// PickPeer picks a peer according to key
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 确保p.peers已经初始化
	if p.peers == nil {
		log.Println("PickPeer() called but peers not properly initialized")
		return nil, false
	}

	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		// 通过一致性哈希(节点负荷平衡)找到该值(应该)存储的节点
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	} else if peer == p.self {
		p.Log("Pick self %s", peer)
		return nil, false
	}
	return nil, false
}

// PickPeers picks multiple peers for hot spot data backup
func (p *HTTPPool) PickPeers(key string, count int) ([]PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 确保p.peers已经初始化
	if p.peers == nil {
		log.Println("PickPeers() called but peers not properly initialized")
		return nil, false
	}

	// 获取主节点
	mainPeer := p.peers.Get(key)
	if mainPeer == "" || mainPeer == p.self {
		return nil, false
	}

	// 收集所有可用的节点（除了自己）
	var availablePeers []string
	for peer := range p.httpGetters {
		if peer != p.self {
			availablePeers = append(availablePeers, peer)
		}
	}

	// 如果可用节点数量不足，返回所有可用节点
	if len(availablePeers) <= count {
		peers := make([]PeerGetter, 0, len(availablePeers))
		for _, peer := range availablePeers {
			peers = append(peers, p.httpGetters[peer])
		}
		p.Log("Pick all %d available peers for hot spot data", len(peers))
		return peers, len(peers) > 0
	}

	// 否则，选择指定数量的节点（包括主节点）
	peers := make([]PeerGetter, 0, count)
	// 首先添加主节点
	peers = append(peers, p.httpGetters[mainPeer])

	// 随机选择其他节点
	for i := 0; i < count-1 && len(availablePeers) > 0; i++ {
		// 简单随机选择一个节点
		index := rand.Intn(len(availablePeers))
		peer := availablePeers[index]
		// 如果不是主节点，添加到结果中
		if peer != mainPeer {
			peers = append(peers, p.httpGetters[peer])
		}
		// 从可用节点中移除已选择的节点
		availablePeers = append(availablePeers[:index], availablePeers[index+1:]...)
	}

	p.Log("Pick %d peers for hot spot data", len(peers))
	return peers, true
}

var _ PeerPicker = (*HTTPPool)(nil)

type httpGetter struct {
	baseURL string
	client  *http.Client
}

func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	// 构建请求的 URL:http://10.0.0.2:8008/_geecache/<groupname>/<key>
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)

	// res, err := http.Get(u) 
	// 发送 HTTP 请求给该地址的 HTTP 服务端，由ServeHttp来处理
	res, err := h.client.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}

	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

// Set sends a PUT request to store a value for a key in remote peer
func (h *httpGetter) Set(in *pb.Request, out *pb.Response) error {
	// 构建请求的 URL:http://10.0.0.2:8008/_geecache/<groupname>/<key>
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)

	// 将响应数据序列化为protobuf
	body, err := proto.Marshal(out)
	if err != nil {
		return fmt.Errorf("encoding request body: %v", err)
	}

	// 创建PUT请求
	req, err := http.NewRequest(http.MethodPut, u, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	// 发送请求
	// client := &http.Client{}
	// res, err := client.Do(req)
	res, err := h.client.Do(req)

	if err != nil {
		return fmt.Errorf("sending request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	return nil
}

var _ PeerGetter = (*httpGetter)(nil)
