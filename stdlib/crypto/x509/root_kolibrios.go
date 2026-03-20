package x509

import "os"

const certFileEnv = "SSL_CERT_FILE"

func (c *Certificate) systemVerify(opts *VerifyOptions) (chains [][]*Certificate, err error) {
	return nil, nil
}

func loadSystemRoots() (*CertPool, error) {
	roots := NewCertPool()

	file := os.Getenv(certFileEnv)
	if file == "" {
		return roots, nil
	}

	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return roots, nil
		}
		return nil, err
	}

	roots.AppendCertsFromPEM(data)
	return roots, nil
}
