package pki

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// ParseSSHPrivateKey parses the string buffer to a SSH private key.
func ParseSSHPrivateKey(buff []byte) (ssh.Signer, error) {
	return ssh.ParsePrivateKey(buff)
}

// ParseSSHAuthorizedKey parses the string buffer to a SSH certificate.
func ParseSSHAuthorizedKey(buff []byte) (*ssh.Certificate, error) {
	key, _, _, _, err := ssh.ParseAuthorizedKey(buff)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse SSH certificate")
	}

	var cert, ok = key.(*ssh.Certificate)
	if !ok {
		return nil, errors.New("unable to cast public key to SSH Certificate")
	}
	return cert, nil
}
