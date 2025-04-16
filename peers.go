package geecache

import pb "geecache/geecachepb"

// PeerPicker is the interface that must be implemented to locate
// the peer that owns a specific key.
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
	// PickPeers returns multiple peers for hot spot data backup
	PickPeers(key string, count int) ([]PeerGetter, bool)
}

// PeerGetter is the interface that must be implemented by a peer.
type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
	// Set stores a value for a key in remote peer
	Set(in *pb.Request, out *pb.Response) error
}
