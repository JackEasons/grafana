//go:build ignore
// +build ignore

//go:generate go run gen.go

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/codejen"
	corecodegen "github.com/grafana/grafana/pkg/codegen"
	"github.com/grafana/grafana/pkg/cuectx"
	"github.com/grafana/grafana/pkg/plugins/codegen"
	"github.com/grafana/grafana/pkg/plugins/codegen/kindsys"
	"github.com/grafana/thema"
)

var skipPlugins = map[string]bool{
	"canvas":         true,
	"heatmap":        true,
	"heatmap-old":    true,
	"candlestick":    true,
	"state-timeline": true,
	"status-history": true,
	"table":          true,
	"timeseries":     true,
	"influxdb":       true, // plugin.json fails validation (defaultMatchFormat)
	"mixed":          true, // plugin.json fails validation (mixed)
	"opentsdb":       true, // plugin.json fails validation (defaultMatchFormat)
}

const sep = string(filepath.Separator)

func main() {
	if len(os.Args) > 1 {
		log.Fatal(fmt.Errorf("plugin thema code generator does not currently accept any arguments\n, got %q", os.Args))
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(fmt.Errorf("could not get working directory: %s", err))
	}
	grootp := strings.Split(cwd, sep)
	groot := filepath.Join(sep, filepath.Join(grootp[:len(grootp)-3]...))
	rt := cuectx.GrafanaThemaRuntime()

	pluginKindGen := codejen.JennyListWithNamer(func(d *kindsys.PluginDecl) string {
		return d.PluginMeta.Id
	})

	pluginKindGen.Append(
		codegen.PluginTreeListJenny(),
		//adaptToPluginDecl(rt, corecodegen.GoTypesJenny("pkg/tsdb", nil)),
	)

	declParser := kindsys.NewDeclParser(rt)
	decls, err := declParser.Parse(os.DirFS(cwd))
	if err != nil {
		log.Fatalln(fmt.Errorf("parsing plugins in dir failed %s: %s", cwd, err))
	}

	jfs, err := pluginKindGen.GenerateFS(decls...)
	if err != nil {
		log.Fatalln(fmt.Errorf("error writing files to disk: %s", err))
	}

	if _, set := os.LookupEnv("CODEGEN_VERIFY"); set {
		if err = jfs.Verify(context.Background(), groot); err != nil {
			log.Fatal(fmt.Errorf("generated code is out of sync with inputs:\n%s\nrun `make gen-cue` to regenerate", err))
		}
	} else if err = jfs.Write(context.Background(), groot); err != nil {
		log.Fatal(fmt.Errorf("error while writing generated code to disk:\n%s", err))
	}
}

func adaptToPluginDecl(rt *thema.Runtime, jen codejen.OneToOne[*corecodegen.DeclForGen]) codejen.OneToOne[*kindsys.PluginDecl] {
	return codejen.AdaptOneToOne(jen, func(pd *kindsys.PluginDecl) *corecodegen.DeclForGen {
		return pd.DeclForGen
	})
}
