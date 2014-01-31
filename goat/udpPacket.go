package goat

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
)

// udpPacket represents the basic values for a UDP tracker connection
type udpPacket struct {
	ConnID  uint64
	Action  uint32
	TransID []byte
}

// FromBytes creates a udpPacket from a packed byte array
func (u udpPacket) FromBytes(buf []byte) (p udpPacket, err error) {
	// Set up recovery function to catch a panic as an error
	// This will run if we attempt to access an out of bounds index
	defer func() {
		if r := recover(); r != nil {
			p = udpPacket{}
			err = errors.New("failed to create udpPacket from bytes")
		}
	}()

	// Current connection ID (initially handshake, then generated by tracker)
	u.ConnID = binary.BigEndian.Uint64(buf[0:8])
	// Action integer (connect: 0, announce: 1)
	u.Action = binary.BigEndian.Uint32(buf[8:12])
	// Transaction ID, to match between requests
	u.TransID = buf[12:16]

	return u, nil
}

// udpAnnouncePacket represents a tracker announce in the UDP format
type udpAnnouncePacket struct {
	InfoHash   string
	PeerID     string
	Downloaded int64
	Left       int64
	Uploaded   int64
	Event      int64
	IP         int64
	Key        string
	Numwant    int64
	Port       int64
}

// FromBytes creates a udpAnnouncePacket from a packed byte array
func (u udpAnnouncePacket) FromBytes(buf []byte) (p udpAnnouncePacket, err error) {
	// Set up recovery function to catch a panic as an error
	// This will run if we attempt to access an out of bounds index
	defer func() {
		if r := recover(); r != nil {
			p = udpAnnouncePacket{}
			err = errors.New("failed to create udpAnnouncePacket from bytes")
		}
	}()

	// InfoHash
	u.InfoHash = string(buf[16:36])

	// PeerID
	u.PeerID = string(buf[36:56])

	// Downloaded
	t, err := strconv.ParseInt(hex.EncodeToString(buf[56:64]), 16, 64)
	if err != nil {
		return udpAnnouncePacket{}, errUDPInteger
	}
	u.Downloaded = t

	// Left
	t, err = strconv.ParseInt(hex.EncodeToString(buf[64:72]), 16, 64)
	if err != nil {
		return udpAnnouncePacket{}, errUDPInteger
	}
	u.Left = t

	// Uploaded
	t, err = strconv.ParseInt(hex.EncodeToString(buf[72:80]), 16, 64)
	if err != nil {
		return udpAnnouncePacket{}, errUDPInteger
	}
	u.Uploaded = t

	// Event
	t, err = strconv.ParseInt(hex.EncodeToString(buf[80:84]), 16, 32)
	if err != nil {
		return udpAnnouncePacket{}, errUDPInteger
	}
	u.Event = t

	// IP address
	t, err = strconv.ParseInt(hex.EncodeToString(buf[84:88]), 16, 32)
	if err != nil {
		return udpAnnouncePacket{}, errUDPInteger
	}
	u.IP = t

	// Key
	u.Key = hex.EncodeToString(buf[88:92])

	// Numwant
	numwant := hex.EncodeToString(buf[92:96])
	// If numwant is hex max value, default to 50
	if numwant == "ffffffff" {
		u.Numwant = 50
	} else {
		t, err = strconv.ParseInt(numwant, 16, 32)
		if err != nil {
			return udpAnnouncePacket{}, errUDPInteger
		}
		u.Numwant = t
	}

	// Port
	t, err = strconv.ParseInt(hex.EncodeToString(buf[96:98]), 16, 32)
	if err != nil {
		return udpAnnouncePacket{}, errUDPInteger
	}
	u.Port = t

	return u, nil
}

// ToValues creates a url.Values struct from a udpAnnouncePacket
func (u udpAnnouncePacket) ToValues() url.Values {
	// Initialize query map
	query := url.Values{}
	query.Set("udp", "1")

	// Copy all fields into query map
	query.Set("info_hash", u.InfoHash)

	// Integer fields
	query.Set("downloaded", strconv.FormatInt(u.Downloaded, 10))
	query.Set("left", strconv.FormatInt(u.Left, 10))
	query.Set("uploaded", strconv.FormatInt(u.Uploaded, 10))

	// Event, converted to actual string
	switch u.Event {
	case 0:
		query.Set("event", "")
	case 1:
		query.Set("event", "completed")
	case 2:
		query.Set("event", "started")
	case 3:
		query.Set("event", "stopped")
	}

	// IP
	query.Set("ip", strconv.FormatInt(u.IP, 10))

	// Key
	query.Set("key", u.Key)

	// Numwant
	query.Set("numwant", strconv.FormatInt(u.Numwant, 10))

	// Port
	query.Set("port", strconv.FormatInt(u.Port, 10))

	// Return final query map
	return query
}

// udpAnnounceResponsePacket represents a tracker announce response in the UDP format
type udpAnnounceResponsePacket struct {
	Action   uint32
	TransID  []byte
	Interval uint32
	Leechers uint32
	Seeders  uint32
	PeerList []compactPeer
}

// FromBytes creates a udpAnnounceResponsePacket from a packed byte array
func (u udpAnnounceResponsePacket) FromBytes(buf []byte) (p udpAnnounceResponsePacket, err error) {
	// Set up recovery function to catch a panic as an error
	// This will run if we attempt to access an out of bounds index
	defer func() {
		if r := recover(); r != nil {
			p = udpAnnounceResponsePacket{}
			err = errors.New("failed to create udpAnnounceResponsePacket from bytes")
		}
	}()

	// Action
	u.Action = binary.BigEndian.Uint32(buf[0:4])

	// Transaction ID
	u.TransID = buf[4:8]

	// Interval
	u.Interval = binary.BigEndian.Uint32(buf[8:12])

	// Leechers
	u.Leechers = binary.BigEndian.Uint32(buf[12:16])

	// Seeders
	u.Seeders = binary.BigEndian.Uint32(buf[16:20])

	// Peer List
	u.PeerList = make([]compactPeer, 0)

	// Iterate peers buffer
	i := 20
	for {
		// Validate that we are not seeking beyond buffer
		if i >= len(buf) {
			break
		}

		// Append peer
		u.PeerList = append(u.PeerList[:], b2ip(buf[i:i+6]))
		i += 6
	}

	return u, nil
}

// udpScrapePacket represents a tracker scrape in the UDP format
type udpScrapePacket struct {
	InfoHashes []string
}

// FromBytes creates a udpScrapePacket from a packed byte array
func (u udpScrapePacket) FromBytes(buf []byte) (p udpScrapePacket, err error) {
	// Set up recovery function to catch a panic as an error
	// This will run if we attempt to access an out of bounds index
	defer func() {
		if r := recover(); r != nil {
			p = udpScrapePacket{}
			err = errors.New("failed to create udpScrapePacket from bytes")
		}
	}()

	// Begin gathering info hashes
	u.InfoHashes = make([]string, 0)

	// Loop and iterate info_hash, up to 70 total (74 is said to be max by BEP15)
	for i := 16; i < 16+(70*20); i += 20 {
		// Validate that we are not appending nil bytes
		if buf[i] == byte(0) {
			break
		}

		u.InfoHashes = append(u.InfoHashes[:], string(buf[i:i+20]))
	}

	return u, nil
}

// ToValues creates a url.Values struct from a udpScrapePacket
func (u udpScrapePacket) ToValues() url.Values {
	// Initialize query map
	query := url.Values{}
	query.Set("udp", "1")

	// Copy InfoHashes slice directly into query
	query["info_hash"] = u.InfoHashes

	// Return final query map
	return query
}
