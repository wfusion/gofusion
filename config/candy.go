package config

import (
	"encoding/base64"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/cipher"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/encode"
)

func (r *registry) Debug() (debug bool) {
	if r.appName != "" {
		return r.debug
	}
	_ = r.LoadComponentConfig(ComponentDebug, &debug)
	return
}

func (r *registry) AppName() (name string) {
	if r.appName != "" {
		return r.appName
	}
	_ = r.LoadComponentConfig(ComponentApp, &name)
	return
}

func (r *registry) DI() di.DI { return r.di }

func (r *registry) cryptoConfig() (conf *CryptoConf) {
	conf = new(CryptoConf)
	if err := r.LoadComponentConfig(ComponentCrypto, &conf); err != nil {
		return
	}
	parseCfgFunc := func(c *cryptoConf) {
		if c == nil {
			return
		}
		c.Algorithm = cipher.ParseAlgorithm(c.AlgorithmString)
		c.Mode = cipher.ParseMode(c.ModeString)
		c.Key = utils.Must(base64.StdEncoding.DecodeString(c.KeyBase64))
		c.IV = utils.Must(base64.StdEncoding.DecodeString(c.IVBase64))

		if utils.IsStrPtrNotBlank(c.CompressAlgorithmString) {
			c.CompressAlgorithm = compress.ParseAlgorithm(*c.CompressAlgorithmString)
		}
		if utils.IsStrPtrNotBlank(c.OutputAlgorithmString) {
			c.OutputAlgorithm = encode.ParseAlgorithm(*c.OutputAlgorithmString)
		}
	}

	if conf != nil {
		parseCfgFunc(conf.Config)
		conf.Custom = make(map[string]*cryptoConf)
		for _, c := range conf.Custom {
			parseCfgFunc(c)
		}
	}

	return
}
