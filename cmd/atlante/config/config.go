package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-spatial/atlante/atlante/style"

	"github.com/gdey/errors"
	"github.com/go-spatial/atlante/atlante"
	"github.com/go-spatial/atlante/atlante/config"
	"github.com/go-spatial/atlante/atlante/filestore"
	fsmulti "github.com/go-spatial/atlante/atlante/filestore/multi"
	"github.com/go-spatial/atlante/atlante/grids"
	"github.com/go-spatial/atlante/atlante/notifiers"
	"github.com/go-spatial/tegola/dict"
	"github.com/prometheus/common/log"
)

var (
	// Providers provides the grid providers
	Providers = make(map[string]grids.Provider)
	// FileStores are the files store providers
	FileStores = make(map[string]filestore.Provider)
	a          atlante.Atlante
)

// Provider is a config structure for Grid Providers
type Provider struct {
	dict.Dicter
}

// NameGridProvider implements grids.Config interface
func (pcfg Provider) NameGridProvider(name string) (grids.Provider, error) {

	name = strings.ToLower(name)
	p, ok := Providers[name]
	if !ok {
		return nil, grids.ErrProviderNotRegistered(name)
	}
	return p, nil

}

// Filestore is a config for file stores
type Filestore struct {
	dict.Dicter
}

// FileStoreFor implements the filestore.Config interface
func (fscfg Filestore) FileStoreFor(name string) (filestore.Provider, error) {
	name = strings.ToLower(name)
	p, ok := FileStores[name]
	if !ok {
		return nil, filestore.ErrUnknownProvider(name)
	}
	return p, nil
}

// LoadConfig will create a new Atlante object based on the config provided
func LoadConfig(conf config.Config, dpi int, overrideDPI bool) (*atlante.Atlante, error) {

	var ok bool
	var a atlante.Atlante

	// Notifier
	if conf.Notifier != nil {
		note, err := notifiers.From(notifiers.Config(conf.Notifier))
		if err != nil {
			return nil, errors.String(fmt.Sprintf("notifier: %v", err))
		}
		a.Notifier = note
	}

	// Loop through and load up any global styles
	styles := make([]style.Style, len(conf.Styles))
	for i, s := range conf.Styles {
		styles[i].Name = string(s.Name)
		styles[i].Description = string(s.Desc)
		styles[i].Location = string(s.Loc)
	}
	if err := style.Append(styles...); err != nil {
		return nil, fmt.Errorf("error global styles %w", err)
	}

	// Loop through providers creating a provider type mapping.
	for i, p := range conf.Providers {
		// type is required
		typ, err := p.String("type", nil)
		if err != nil {
			return nil, fmt.Errorf("error provider (%v) missing type : %w", i, err)
		}
		name, err := p.String("name", nil)
		if err != nil {
			return nil, fmt.Errorf("error provider( %v) missing name : %w", i, err)
		}
		name = strings.ToLower(name)
		if _, ok := Providers[name]; ok {
			return nil, fmt.Errorf("error provider with name (%v) is already registered", name)
		}
		prv, err := grids.For(typ, Provider{p})
		if err != nil {
			return nil, fmt.Errorf("error registering provider (%v -- %v)(#%v): %w", typ, name, i, err)
		}

		Providers[name] = prv
		log.Infof("configured grid provider: %v (%v)", name, typ)
	}

	// filestores
	for i, fstore := range conf.FileStores {
		// type is required
		typ, err := fstore.String("type", nil)
		if err != nil {
			return nil, fmt.Errorf("error filestore (%v) missing type : %v", i, err)
		}
		name, err := fstore.String("name", nil)
		if err != nil {
			return nil, fmt.Errorf("error filestore (%v) missing name: %v", i, err)
		}
		name = strings.ToLower(name)
		if _, ok = FileStores[name]; ok {
			return nil, fmt.Errorf("error provider(%v) with name (%v) is already registered", i, name)
		}
		prv, err := filestore.For(typ, Filestore{fstore})
		if err != nil {
			return nil, fmt.Errorf("error registering filestore %v:%v", i, err)
		}
		FileStores[name] = prv
	}

	if len(conf.Sheets) == 0 {
		return nil, fmt.Errorf("no sheets configured")
	}
	// Establish sheets
	for i, sheet := range conf.Sheets {

		providerName := strings.ToLower(string(sheet.ProviderGrid))

		prv, ok := Providers[providerName]
		if providerName != "" && !ok {
			return nil, fmt.Errorf("error locating provider (%v) for sheet %v (#%v)", providerName, sheet.Name, i)
		}
		templateURL, err := url.Parse(string(sheet.Template))
		if err != nil {
			return nil, fmt.Errorf("error parsing template url (%v) for sheet %v (#%v)",
				string(sheet.Template),
				sheet.Name,
				i,
			)
		}
		name := strings.ToLower(string(sheet.Name))
		var fstores []filestore.Provider
		for _, filestoreString := range sheet.Filestores {
			filestoreName := strings.TrimSpace(strings.ToLower(string(filestoreString)))
			var fsprv filestore.Provider
			if filestoreName == "" {
				continue
			}
			fsprv, ok = FileStores[filestoreName]
			if !ok {
				log.Warnln("Known file stores are:")
				for k := range FileStores {
					log.Warnln("\t", k)
				}
				return nil, filestore.ErrUnknownProvider(filestoreName)
			}
			fstores = append(fstores, fsprv)
		}
		var fsprv filestore.Provider
		switch len(fstores) {
		case 0:
			fsprv = nil
		case 1:
			fsprv = fstores[0]
		default:
			fsprv = fsmulti.New(fstores...)
		}
		odpi := uint(sheet.DPI)
		// 0 means it's not set
		if overrideDPI || odpi == 0 {
			odpi = uint(dpi)
		}
		var stylelist = style.Provider((*style.List)(nil))

		{
			// Here we need to check to see if
			// we have multiple styles or the old
			// style entry or both.

			styleEntry := string(sheet.Style)
			stylesEntry := ([]string)(sheet.Styles)
			switch {

			case len(stylesEntry) > 0:
				// we have a styles entry. Always use that
				stylelist = style.SubList(stylesEntry...)

			case styleEntry != "":
				// Old style config. We need to build a new style
				// name and entry and issue a warning.
				styleName := fmt.Sprintf("%s_style", name)
				style.Append(style.Style{
					Name:        styleName,
					Description: fmt.Sprintf("Style for sheet %v:\n%v", name, sheet.Description),
					Location:    styleEntry,
				})
				stylelist = style.SubList(styleName)
				log.Warnf("For sheet %v, style is deprecated, please use styles instead", name)

			default:
				// values
				// No style list is defined just use the global
			}

		}

		sht, err := atlante.NewSheet(
			name,
			prv,
			uint(odpi),
			string(sheet.Description),
			stylelist,
			templateURL,
			fsprv,
		)
		if err != nil {
			return nil, fmt.Errorf("error trying to create sheet %v: %v", i, err)
		}
		if sheet.Height != 0 {
			sht.Height = float64(sheet.Height)
		}
		if sheet.Width != 0 {
			sht.Width = float64(sheet.Width)
		}

		err = a.AddSheet(sht)
		if err != nil {
			return nil, fmt.Errorf("error trying to add sheet %v: %v", i, err)
		}
	}
	return &a, nil

}

// Load will attempt to load and validate a config at the given location
func Load(location string, dpi int, overrideDPI bool) (*atlante.Atlante, error) {

	aURL, err := url.Parse(location)
	if err != nil {
		return nil, err
	}
	conf, err := config.LoadAndValidate(aURL)
	if err != nil {
		return nil, err
	}
	return LoadConfig(conf, dpi, overrideDPI)
}
