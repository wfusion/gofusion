package config

import (
	"encoding/base64"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/cipher"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/encode"
)

func (p *registry) Debug() (debug bool) {
	if p.appName != "" {
		return p.debug
	}
	_ = p.LoadComponentConfig(ComponentDebug, &debug)
	return
}

func (p *registry) AppName() (name string) {
	if p.appName != "" {
		return p.appName
	}
	_ = p.LoadComponentConfig(ComponentApp, &name)
	return
}

func (p *registry) DI() di.DI { return p.di }

func (p *registry) cryptoConfig() (conf *CryptoConf) {
	conf = new(CryptoConf)
	if err := p.LoadComponentConfig(ComponentCrypto, &conf); err != nil {
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
