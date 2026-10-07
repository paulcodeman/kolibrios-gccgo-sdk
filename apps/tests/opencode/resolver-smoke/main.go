package main

import (
	"context"
	"errors"
	"io"
	"kos"
	"net"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

func check(ok bool, why string) {
	if !ok { kos.DebugString(why); panic(why) }
}
func must(err error) { if err != nil { kos.DebugString(err.Error()); panic(err) } }

func answer(connection net.Conn) {
	defer connection.Close()
	must(connection.SetDeadline(time.Now().Add(5*time.Second)))
	var length [2]byte
	_, err := io.ReadFull(connection, length[:]); must(err)
	request := make([]byte, int(length[0])<<8|int(length[1]))
	_, err = io.ReadFull(connection, request); must(err)
	var parser dnsmessage.Parser
	header, err := parser.Start(request); must(err)
	question, err := parser.Question(); must(err)
	response := dnsmessage.Header{ID: header.ID, Response: true, RecursionAvailable: true}
	if question.Name.String() == "missing.test." { response.RCode = dnsmessage.RCodeNameError }
	if question.Name.String() == "partial.test." && question.Type == dnsmessage.TypeAAAA { response.RCode = dnsmessage.RCodeServerFailure }
	builder := dnsmessage.NewBuilder(nil, response)
	builder.EnableCompression()
	must(builder.StartQuestions()); must(builder.Question(question)); must(builder.StartAnswers())
	h := dnsmessage.ResourceHeader{Name: question.Name, Class: dnsmessage.ClassINET, TTL: 60}
	if response.RCode == dnsmessage.RCodeSuccess {
		switch question.Type {
		case dnsmessage.TypeA:
			must(builder.AResource(h, dnsmessage.AResource{A: [4]byte{127,0,0,42}}))
		case dnsmessage.TypeAAAA:
			must(builder.AAAAResource(h, dnsmessage.AAAAResource{AAAA: [16]byte{15:1}}))
		case dnsmessage.TypeTXT:
			must(builder.TXTResource(h, dnsmessage.TXTResource{TXT: []string{"grpc", "-config"}}))
		case dnsmessage.TypeSRV:
			target, err := dnsmessage.NewName("backend.test."); must(err)
			must(builder.SRVResource(h, dnsmessage.SRVResource{Target: target, Port: 443, Priority: 20, Weight: 1}))
			must(builder.SRVResource(h, dnsmessage.SRVResource{Target: target, Port: 8443, Priority: 10, Weight: 1}))
		}
	}
	data, err := builder.Finish(); must(err)
	length[0], length[1] = byte(len(data)>>8), byte(len(data))
	_, err = connection.Write(append(length[:], data...)); must(err)
}

func main() {
	console, ok := kos.OpenConsole("OpenCode DNS resolver test"); check(ok,"console")
	defer console.Close()
	kos.DebugString("OPENCODE_RESOLVER_START")
	listener, err := net.Listen("tcp", "127.0.0.1:0"); must(err)
	defer listener.Close()
	go func() {
		for { connection, err := listener.Accept(); if err != nil { return }; go answer(connection) }
	}()
	resolver := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		check(network == "tcp", "DNS uses real framed TCP")
		check(net.ParseIP("127.0.0.1") != nil && address != "", "DNS dial literal address")
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, listener.Addr().String())
	}}
	ctx := context.Background()
	addresses, err := resolver.LookupHost(ctx, "host.test"); must(err)
	check(len(addresses)==2 && addresses[0]=="127.0.0.42" && addresses[1]=="::1", "DNS A and AAAA decoding")
	txt, err := resolver.LookupTXT(ctx, "config.test"); must(err)
	check(len(txt)==1 && txt[0]=="grpc-config", "DNS TXT segments concatenate")
	cname, srv, err := resolver.LookupSRV(ctx, "grpclb", "tcp", "service.test"); must(err)
	check(cname=="_grpclb._tcp.service.test." && len(srv)==2 && srv[0].Port==8443 && srv[1].Port==443 && srv[0].Target=="backend.test.", "DNS SRV original priority ordering")
	_, err = resolver.LookupTXT(ctx, "missing.test")
	var dns *net.DNSError
	check(errors.As(err,&dns) && dns.IsNotFound && !dns.IsTimeout, "DNS NXDOMAIN classification")
	addresses, err = resolver.LookupHost(ctx, "partial.test"); must(err)
	check(len(addresses)==1 && addresses[0]=="127.0.0.42", "DNS partial address lookup")
	resolver.StrictErrors = true
	_, err = resolver.LookupHost(ctx, "partial.test")
	check(errors.As(err,&dns) && dns.IsTemporary, "DNS StrictErrors rejects SERVFAIL")
	_, err = resolver.LookupHost(ctx, "")
	check(errors.As(err,&dns) && dns.IsNotFound, "DNS empty host rejected")
	canceled, cancel := context.WithCancel(ctx); cancel()
	_, err = resolver.LookupTXT(canceled,"config.test")
	check(err != nil, "DNS canceled context rejected")
	kos.DebugString("OPENCODE_RESOLVER_PASS")
	console.WriteString("OPENCODE_RESOLVER_PASS\n")
}
