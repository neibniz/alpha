package config

import (
	"errors"
	"strings"
)

const minimumSigningSecretBytes = 32

func ValidateRuntimeSecurity(cfg Config) error {
	secret := strings.TrimSpace(cfg.JwtAuth.AccessSecret)
	if len(secret) < minimumSigningSecretBytes {
		return errors.New("JwtAuth.AccessSecret must contain at least 32 bytes")
	}
	nodeSecret := strings.TrimSpace(cfg.Node.NodeSecret)
	deploySecret := strings.TrimSpace(cfg.Node.NodeDeploySecret)
	if (nodeSecret == "") != (deploySecret == "") {
		return errors.New("Node.NodeSecret and Node.NodeDeploySecret must be configured together")
	}
	if nodeSecret != "" && len(nodeSecret) < minimumSigningSecretBytes {
		return errors.New("Node.NodeSecret must contain at least 32 bytes")
	}
	if deploySecret != "" && len(deploySecret) < minimumSigningSecretBytes {
		return errors.New("Node.NodeDeploySecret must contain at least 32 bytes")
	}
	forwarderSecret := strings.TrimSpace(cfg.Forwarder.BootstrapSecret)
	if forwarderSecret != "" && len(forwarderSecret) < minimumSigningSecretBytes {
		return errors.New("Forwarder.BootstrapSecret must contain at least 32 bytes")
	}
	return nil
}
