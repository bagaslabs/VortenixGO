package enet

/*
#cgo CFLAGS: -I.
#cgo !windows LDFLAGS: -Lenet/ -lenet -lstdc++
#cgo windows LDFLAGS: -lWs2_32 -lWinmm -lstdc++

#include <winsock2.h>
#include <windows.h>
#include "enet.h"
#include <stdlib.h>
#include <time.h>

// Helper to set checksum
void enet_host_set_checksum_crc32(ENetHost* host) {
    host->checksum = enet_crc32;
}

// Helper to set usingNewPacket
void enet_host_set_using_new_packet(ENetHost* host, int val) {
    host->usingNewPacket = val;
}

// Helper to set proxy
void enet_host_use_proxy_helper(ENetHost* host, const char* ip, int port, const char* user, const char* pass) {
    enet_host_use_proxy(host, ip, (uint16_t)port, user, pass);
}

// Helper to set max packet size and waiting data
void enet_host_set_max_packet_limits(ENetHost* host, size_t size) {
    host->maximumPacketSize = size;
    host->maximumWaitingData = size * 2;
}

// Helper to set socket buffers
void enet_host_set_socket_buffers(ENetHost* host, size_t rcv, size_t snd) {
    enet_socket_set_option(host->socket, ENET_SOCKOPT_RCVBUF, rcv);
    enet_socket_set_option(host->socket, ENET_SOCKOPT_SNDBUF, snd);
}

// Global initialization helper
void enet_initialize_helper() {
    enet_initialize();
    srand((unsigned int)time(NULL));
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// Initialize enet
func Initialize() int {
	C.enet_initialize_helper()
	return 0
}

// Deinitialize enet
func Deinitialize() {
	C.enet_deinitialize()
}

// LinkedVersion returns the linked version of enet currently being used.
// Returns MAJOR.MINOR.PATCH as a string.
func LinkedVersion() string {
	var version = uint32(C.enet_linked_version())
	major := uint8(version >> 16)
	minor := uint8(version >> 8)
	patch := uint8(version)
	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}

// Host wraps ENetHost
type Host struct {
	cHost *C.ENetHost
}

// Peer wraps ENetPeer
type Peer struct {
	cPeer *C.ENetPeer
}

// Address wraps ENetAddress
type Address struct {
	cAddr C.ENetAddress
}

func NewAddress(host string, port int) (*Address, error) {
	addr := &Address{}
	addr.cAddr.port = C.enet_uint16(port)

	cHost := C.CString(host)
	defer C.free(unsafe.Pointer(cHost))

	if C.enet_address_set_host(&addr.cAddr, cHost) != 0 {
		return nil, fmt.Errorf("failed to set host address")
	}
	return addr, nil
}

func CreateHost(address *Address, peerCount int, channelLimit int, incomingBandwidth uint32, outgoingBandwidth uint32) *Host {
	var cAddr *C.ENetAddress
	if address != nil {
		cAddr = &address.cAddr
	}
	host := C.enet_host_create(cAddr, C.size_t(peerCount), C.size_t(channelLimit), C.enet_uint32(incomingBandwidth), C.enet_uint32(outgoingBandwidth))
	if host == nil {
		return nil
	}
	return &Host{cHost: host}
}

func (h *Host) Destroy() {
	if h.cHost != nil {
		C.enet_host_destroy(h.cHost)
		h.cHost = nil
	}
}

func (h *Host) CompressWithRangeCoder() {
	if h.cHost != nil {
		C.enet_host_compress_with_range_coder(h.cHost)
	}
}

func (h *Host) SetChecksum() {
	if h.cHost != nil {
		C.enet_host_set_checksum_crc32(h.cHost)
	}
}

func (h *Host) SetUsingNewPacket(value bool) {
	if h.cHost != nil {
		val := 0
		if value {
			val = 1
		}
		C.enet_host_set_using_new_packet(h.cHost, C.int(val))
	}
}

func (h *Host) SetProxy(ip string, port int, user string, pass string) {
	if h.cHost != nil {
		cip := C.CString(ip)
		cuser := C.CString(user)
		cpass := C.CString(pass)
		defer C.free(unsafe.Pointer(cip))
		defer C.free(unsafe.Pointer(cuser))
		defer C.free(unsafe.Pointer(cpass))
		C.enet_host_use_proxy_helper(h.cHost, cip, C.int(port), cuser, cpass)
	}
}

func (h *Host) SetMaxPacketLimits(size int) {
	if h.cHost != nil {
		C.enet_host_set_max_packet_limits(h.cHost, C.size_t(size))
	}
}

func (h *Host) SetSocketBuffers(rcv, snd int) {
	if h.cHost != nil {
		C.enet_host_set_socket_buffers(h.cHost, C.size_t(rcv), C.size_t(snd))
	}
}

func (h *Host) Connect(address *Address, channelCount int, data uint32) *Peer {
	if h.cHost == nil || address == nil {
		return nil
	}
	peer := C.enet_host_connect(h.cHost, &address.cAddr, C.size_t(channelCount), C.enet_uint32(data))
	if peer == nil {
		return nil
	}
	return &Peer{cPeer: peer}
}

func (h *Host) Flush() {
	if h.cHost != nil {
		C.enet_host_flush(h.cHost)
	}
}

func (p *Peer) Send(channelID uint8, data []byte, flags uint32) int {
	if p.cPeer == nil {
		return -1
	}
	var ptr unsafe.Pointer
	if len(data) > 0 {
		ptr = unsafe.Pointer(&data[0])
	}
	packet := C.enet_packet_create(ptr, C.size_t(len(data)), C.enet_uint32(flags))
	return int(C.enet_peer_send(p.cPeer, C.enet_uint8(channelID), packet))
}

func (p *Peer) Disconnect(data uint32) {
	if p.cPeer != nil {
		C.enet_peer_disconnect(p.cPeer, C.enet_uint32(data))
	}
}

func (p *Peer) Reset() {
	if p.cPeer != nil {
		C.enet_peer_reset(p.cPeer)
		p.cPeer = nil
	}
}

func (p *Peer) GetRoundTripTime() int {
	if p.cPeer != nil {
		return int(p.cPeer.roundTripTime)
	}
	return 0
}

func (p *Peer) IsNil() bool {
	return p.cPeer == nil
}

// Packet wraps ENetPacket
type Packet struct {
	cPacket *C.ENetPacket
}

func (p *Packet) Destroy() {
	if p.cPacket != nil {
		C.enet_packet_destroy(p.cPacket)
		p.cPacket = nil
	}
}

func (p *Packet) GetData() []byte {
	if p.cPacket == nil || p.cPacket.data == nil {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(p.cPacket.data), C.int(p.cPacket.dataLength))
}

func (p *Packet) GetLength() int {
	if p.cPacket == nil {
		return 0
	}
	return int(p.cPacket.dataLength)
}

type EventType int

const (
	EventNone       EventType = C.ENET_EVENT_TYPE_NONE
	EventConnect    EventType = C.ENET_EVENT_TYPE_CONNECT
	EventDisconnect EventType = C.ENET_EVENT_TYPE_DISCONNECT
	EventReceive    EventType = C.ENET_EVENT_TYPE_RECEIVE
)

type Event struct {
	Type      EventType
	Peer      *Peer
	ChannelID uint8
	Data      uint32
	Packet    *Packet
}

func (h *Host) Service(timeout int) (*Event, error) {
	if h.cHost == nil {
		return nil, fmt.Errorf("host is nil")
	}
	var event C.ENetEvent
	ret := C.enet_host_service(h.cHost, &event, C.enet_uint32(timeout))
	if ret < 0 {
		return nil, fmt.Errorf("service error")
	}
	if ret == 0 {
		return nil, nil
	}

	ev := &Event{
		Type:      EventType(event._type),
		ChannelID: uint8(event.channelID),
		Data:      uint32(event.data),
	}
	if event.peer != nil {
		ev.Peer = &Peer{cPeer: event.peer}
	}
	if event.packet != nil {
		ev.Packet = &Packet{cPacket: event.packet}
	}

	return ev, nil
}

func (p *Peer) GetRemoteIP() string {
	if p.cPeer == nil {
		return ""
	}
	var ip [64]C.char
	C.enet_address_get_host_ip(&p.cPeer.address, &ip[0], 64)
	return C.GoString(&ip[0])
}

func (p *Peer) GetRemotePort() int {
	if p.cPeer == nil {
		return 0
	}
	return int(p.cPeer.address.port)
}
