package main

import (
	cfg "github.com/conductorone/baton-coupa/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("coupa", cfg.Config)
}
