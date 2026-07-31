package config

import "testing"

func TestValidateRuntimeSecurityRejectsWeakJWTSecret(t *testing.T) {
	if err := ValidateRuntimeSecurity(Config{JwtAuth: JwtAuth{AccessSecret: "short"}}); err == nil {
		t.Fatal("weak JWT secret should be rejected")
	}
	if err := ValidateRuntimeSecurity(Config{JwtAuth: JwtAuth{AccessSecret: "0123456789abcdef0123456789abcdef"}}); err != nil {
		t.Fatalf("strong JWT secret rejected: %v", err)
	}
}

func TestValidateRuntimeSecurityRejectsWeakOrPartialNodeSecrets(t *testing.T) {
	const strong = "0123456789abcdef0123456789abcdef"
	tests := []Config{
		{JwtAuth: JwtAuth{AccessSecret: strong}, Node: NodeConfig{NodeSecret: "short", NodeDeploySecret: strong}},
		{JwtAuth: JwtAuth{AccessSecret: strong}, Node: NodeConfig{NodeSecret: strong, NodeDeploySecret: "short"}},
		{JwtAuth: JwtAuth{AccessSecret: strong}, Node: NodeConfig{NodeSecret: strong}},
		{JwtAuth: JwtAuth{AccessSecret: strong}, Node: NodeConfig{NodeDeploySecret: strong}},
	}
	for index, cfg := range tests {
		if err := ValidateRuntimeSecurity(cfg); err == nil {
			t.Fatalf("case %d: invalid node secrets should be rejected", index)
		}
	}
}

func TestValidateRuntimeSecurityAllowsDisabledOrStrongNodeSecrets(t *testing.T) {
	const strong = "0123456789abcdef0123456789abcdef"
	for index, cfg := range []Config{
		{JwtAuth: JwtAuth{AccessSecret: strong}},
		{JwtAuth: JwtAuth{AccessSecret: strong}, Node: NodeConfig{NodeSecret: strong, NodeDeploySecret: strong}},
	} {
		if err := ValidateRuntimeSecurity(cfg); err != nil {
			t.Fatalf("case %d: valid security config rejected: %v", index, err)
		}
	}
}
