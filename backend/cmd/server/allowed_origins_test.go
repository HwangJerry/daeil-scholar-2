package main

import (
	"reflect"
	"testing"

	"github.com/dflh-saf/backend/internal/config"
)

func TestConfiguredOriginsAddsAdminOrigin(t *testing.T) {
	got := configuredOrigins(config.ServerConfig{
		AllowedOrigin: "https://daeilfoundation.or.kr",
		AdminOrigin:   " https://adms.daeilfoundation.or.kr ",
	})
	want := []string{"https://daeilfoundation.or.kr", "https://adms.daeilfoundation.or.kr"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("origins = %v, want %v", got, want)
	}
}

func TestConfiguredOriginsWithoutAdminOrigin(t *testing.T) {
	got := configuredOrigins(config.ServerConfig{AllowedOrigin: "https://a.example, https://b.example"})
	want := []string{"https://a.example", "https://b.example"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("origins = %v, want %v", got, want)
	}
}
