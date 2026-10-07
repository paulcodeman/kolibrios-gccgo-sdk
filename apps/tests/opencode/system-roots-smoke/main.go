package main

import (
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"kos"
	"net/http"
	"os"
	"time"
)

func main() {
	console, ok := kos.OpenConsole("Upstream system certificate store")
	if !ok { panic("console initialization") }
	defer console.Exit(false)
	roots, err := x509.SystemCertPool()
	if err != nil || len(roots.Subjects()) < 121 { panic("default Mozilla system roots") }
	fmt.Println("Loaded default certificate roots:", len(roots.Subjects()))
	client := &http.Client{Timeout: 30*time.Second}
	response, err := client.Get("https://10.0.2.2:18443/health")
	if os.Getenv("KOLIBRI_TEST_EXPECT_UNTRUSTED") == "1" {
		var untrusted x509.UnknownAuthorityError
		if !errors.As(err, &untrusted) { panic("untrusted certificate was not rejected") }
		kos.DebugString("OPENCODE_SYSTEM_ROOTS_UNTRUSTED_PASS\n")
		return
	}
	if err != nil { fmt.Println(err); panic("default root directory TLS") }
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != 200 || len(body)==0 || response.TLS==nil || len(response.TLS.VerifiedChains)==0 { panic("verified system root HTTPS") }
	fmt.Println("Default roots and verified HTTPS PASS")
	kos.DebugString("OPENCODE_SYSTEM_ROOTS_PASS\n")
}
