package server

import (
    "fmt"
    "strconv"
    "net"
)

const big = 0xFFFFFF
func dtoi(s string, i0 int) (n int, i int, ok bool) {
    n = 0
    for i = i0; i < len(s) && '0' <= s[i] && s[i] <= '9'; i++ {
        n = n*10 + int(s[i]-'0')
        if n >= big {
            return 0, i, false
        }
    }
    if i == i0 {
        return 0, i, false
    }
    return n, i, true
}


func parseIPFmt(s string) ([]byte) {
    var p [4]byte
    i := 0
    for j := 0; j < 4; j++ {
        if i >= len(s) {
            // Missing octets.
            return nil
        }
        if j > 0 {
            if s[i] != '.' {
                return nil
            }
            i++
        }
        var (
            n  int
            ok bool
        )
        n, i, ok = dtoi(s, i)
        if !ok || n > 0xFF {
            return nil
        }
        p[j] = byte(n)
    }
    if i != len(s) {
        return nil
    }
    return []byte {p[0], p[1], p[2], p[3]}
}

func parseIntFmt(str string) ([]byte) {
    i, err := strconv.ParseUint(str, 10, 32)
    if err != nil {
        return nil
    }
    var p [4]byte
    fmt.Println(i)
    p[0] = byte(i & 0xFF)
    p[1] = byte((i & 0xFF00) >> 8)
    p[2] = byte((i & 0xFF0000) >> 16)
    p[3] = byte((i & 0xFF000000) >> 24)
    return []byte {p[0], p[1], p[2], p[3]}
}

func convertAreaOrRouterId(str string) ([]byte) {
    for i := 0; i < len(str); i++ {
        if str[i] == '.' {
            return parseIPFmt(str)
        }
    }
    return parseIntFmt(str)
}

func convertAuthKey(s string) ([]byte) {
    var p [8]byte
    i := 0
    for j := 0; j < 8; j++ {
        if i >= len(s) {
            // Missing octets.
            return nil
        }
        if j > 0 {
            if s[i] != '.' {
                return nil
            }
            i++
        }
        var (
            n  int
            ok bool
        )
        n, i, ok = dtoi(s, i)
        if !ok || n > 0xFF {
            return nil
        }
        p[j] = byte(n)
    }
    if i != len(s) {
        return nil
    }
    return []byte {p[0], p[1], p[2], p[3], p[4], p[5], p[6], p[7]}
}

func computeCheckSum(pkt []byte) (uint16) {
    var csum uint32

    for i := 0; i < len(pkt); i+= 2 {
        csum += uint32(pkt[i]) << 8
        csum += uint32(pkt[i+1])
    }
    chkSum := ^uint16((csum >> 16) + csum)
    return chkSum
}

func bytesEqual(x, y []byte) bool {
    if len(x) != len(y) {
        return false
    }
    for i, b := range x {
        if y[i] != b {
            return false
        }
    }
    return true
}

func isInSubnet(ifIpAddr net.IP, srcIp net.IP, netMask net.IPMask) (bool) {
    net1 := ifIpAddr.Mask(netMask)
    net2 := srcIp.Mask(netMask)
    if net1.Equal(net2) {
        return true
    }
    return false
}

func convertIPv4ToUint32(ip []byte) (uint32) {
    var val uint32 = 0

    val =  val + uint32(ip[0])
    val = (val << 8) + uint32(ip[1])
    val = (val << 8) + uint32(ip[2])
    val = (val << 8) + uint32(ip[3])

    return val
}