package utils

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"path/filepath"
)

func TlsConfigFromDir(certDir string) (*tls.Config, error) {
	certBytes, err := ioutil.ReadFile(filepath.Join(certDir, "cert-chain.pem"))
	if err != nil {
		return nil, err
	}

	keyBytes, err := ioutil.ReadFile(filepath.Join(certDir, "key.pem"))
	if err != nil {
		return nil, err
	}

	caBytes, err := ioutil.ReadFile(filepath.Join(certDir, "root-cert.pem"))
	if err != nil {
		return nil, err
	}

	return TlsConfig(certBytes, keyBytes, caBytes)
}

func TlsConfig(certBytes, keyBytes, caBytes []byte) (*tls.Config, error) {
	clientCert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return nil, err
	}

	serverCAs := x509.NewCertPool()
	if ok := serverCAs.AppendCertsFromPEM(caBytes); !ok {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      serverCAs,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			return nil
		},
	}, nil
}
