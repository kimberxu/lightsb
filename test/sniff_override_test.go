package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/netip"
	"strconv"
	"testing"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/json/badoption"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/protocol/socks"

	"github.com/stretchr/testify/require"
)

// startTLSServer listens on 127.0.0.1:testPort with the given certificate and
// replies with a fixed payload after a successful handshake.
func startTLSServer(t *testing.T, certPath string, keyPath string) {
	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	require.NoError(t, err)
	listener, err := tls.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(int(testPort))), &tls.Config{
		Certificates: []tls.Certificate{certificate},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		listener.Close()
	})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				_, _ = conn.Write([]byte("pong"))
				conn.Close()
			}()
		}
	}()
}

// TestSniffOverrideDestination covers the fork-only action.sniff.override_destination
// field: the sniffed SNI replaces the destination before the following rules match.
//
// Client dials the literal address 127.0.0.1:testPort, but sends SNI "localhost".
// With override_destination enabled the destination becomes localhost:testPort and
// the trailing ip_cidr rule no longer matches, so the connection reaches the TLS
// server. Without it the destination stays 127.0.0.1 and the reject rule closes it.
func TestSniffOverrideDestination(t *testing.T) {
	_, certPath, keyPath := createSelfSignedCertificate(t, "localhost")
	startTLSServer(t, certPath, keyPath)

	startInstance(t, option.Options{
		Inbounds: []option.Inbound{
			{
				Type: C.TypeMixed,
				Tag:  "mixed-in",
				Options: &option.HTTPMixedInboundOptions{
					ListenOptions: option.ListenOptions{
						Listen:     common.Ptr(badoption.Addr(netip.IPv4Unspecified())),
						ListenPort: clientPort,
					},
				},
			},
		},
		Outbounds: []option.Outbound{
			{
				Type: C.TypeDirect,
			},
		},
		Route: &option.RouteOptions{
			Rules: []option.Rule{
				{
					Type: C.RuleTypeDefault,
					DefaultOptions: option.DefaultRule{
						RuleAction: option.RuleAction{
							Action: C.RuleActionTypeSniff,
							SniffOptions: option.RouteActionSniff{
								Sniffer:             []string{C.ProtocolTLS},
								OverrideDestination: true,
							},
						},
					},
				},
				{
					Type: C.RuleTypeDefault,
					DefaultOptions: option.DefaultRule{
						RawDefaultRule: option.RawDefaultRule{
							IPCIDR: []string{"127.0.0.1/32"},
						},
						RuleAction: option.RuleAction{
							Action: C.RuleActionTypeReject,
							// Programmatic option.Options construction bypasses
							// RejectActionOptions.UnmarshalJSON, which normally fills
							// this default in.
							RejectOptions: option.RejectActionOptions{
								Method: C.RuleActionRejectMethodDefault,
							},
						},
					},
				},
			},
		},
	})

	dialer := socks.NewClient(N.SystemDialer, M.ParseSocksaddrHostPort("127.0.0.1", clientPort), socks.Version5, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rawConn, err := dialer.DialContext(ctx, "tcp", M.ParseSocksaddrHostPort("127.0.0.1", testPort))
	require.NoError(t, err)
	defer rawConn.Close()

	tlsConn := tls.Client(rawConn, &tls.Config{
		ServerName:         "localhost",
		InsecureSkipVerify: true,
	})
	require.NoError(t, tlsConn.HandshakeContext(ctx))

	response := make([]byte, 4)
	tlsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := tlsConn.Read(response)
	require.NoError(t, err)
	require.Equal(t, "pong", string(response[:n]))
}
