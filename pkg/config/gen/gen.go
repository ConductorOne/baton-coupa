package main

import (
	"github.com/conductorone/baton-sdk/pkg/config"
	cfg "github.com/conductorone/baton-coupa/pkg/config"
)

func main() {
	config.Generate("coupa", cfg.Config)
}
