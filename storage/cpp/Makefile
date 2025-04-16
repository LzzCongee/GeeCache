CXX = g++
CXXFLAGS = -std=c++11 -Wall -Wextra -fPIC
LDFLAGS = -shared

SRCS = memory_storage.cpp storage_factory.cpp storage_wrapper.cpp
OBJS = $(SRCS:.cpp=.o)
TARGET = libstorage.so

.PHONY: all clean

all: $(TARGET)

$(TARGET): $(OBJS)
	$(CXX) $(LDFLAGS) -o $@ $^

%.o: %.cpp
	$(CXX) $(CXXFLAGS) -c $< -o $@

clean:
	rm -f $(OBJS) $(TARGET)
 