package config

import (
	"io"
	_ "unsafe"

	"github.com/spf13/viper"
)

//go:linkname viperUnmarshalReader github.com/spf13/viper.(*Viper).unmarshalReader
func viperUnmarshalReader(v *viper.Viper, in io.Reader, c map[string]interface{}) error
