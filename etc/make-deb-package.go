package main

import (
	"log"
	"os"

	"github.com/goreleaser/nfpm"
	_ "github.com/goreleaser/nfpm/deb"
)

// should be filled by go build
var Version = "0.0.0-devel"

func defaultInfo() nfpm.Info {
	return nfpm.WithDefaults(nfpm.Info{
		Name:        "nginx-log-collector",
		Arch:        "amd64",
		Description: "Collects nginx logs from rsyslog and saves them into clickhouse dbms",
		Maintainer:  "Oleg Matrokhin <oyumatrokhin@avito.ru>",
		Version:     Version,
		Overridables: nfpm.Overridables{
			Files: map[string]string{
				"./build/nginx-log-collector": "/usr/bin/nginx-log-collector",
			},
			EmptyFolders: []string{
				"/var/lib/nginx-log-collector/backlog/",
				"/var/log/nginx-log-collector/",
			},
			ConfigFiles: map[string]string{
				"./etc/config.yaml":                 "/etc/nginx-log-collector/config.yaml",
				"./etc/nginx-log-collector.service": "/lib/systemd/system/nginx-log-collector.service",
			},
			Scripts: nfpm.Scripts{
				PostInstall: "./etc/debian/postinst.sh",
				PreRemove:   "./etc/debian/prerm.sh",
			},
		},
	})
}

func main() {
	pkg, err := nfpm.Get("deb")
	if err != nil {
		log.Fatalln(err)
	}
	i := defaultInfo()

	f, err := os.Create("./build/nginx-log-collector-" + Version + ".deb")
	if err != nil {
		log.Fatalln(err)
	}

	pkg.Package(i, f)
	f.Close()
}
