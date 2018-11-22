package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/codegangsta/cli"
	"github.com/denverdino/aliyungo/dns"
	"github.com/gutengo/fil"
	"github.com/gutengo/shell"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
)

var pd = fmt.Println

type Rc struct {
	ACCESS_KEY_ID     string
	ACCESS_KEY_SECRET string
}

var rc Rc

func main() {
	cli.AppHelpTemplate = `{{.Name}} v{{.Version}} - {{.Usage}}

USAGE:
   {{.Name}} {{if .Flags}}[options] {{end}}<command> [arguments]

COMMANDS:
   {{range .Commands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
   {{end}}{{if .Flags}}
GLOBAL OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{end}}
`

	app := cli.NewApp()
	app.Name = "aliyundns"
	app.Usage = "list and update dns record"
	app.Version = "0.0.1"

	app.Action = func(c *cli.Context) {
		cli.ShowAppHelp(c)
	}

	app.Commands = []cli.Command{
		{
			Name:  "list",
			Usage: "<domain>",
			Action: func(c *cli.Context) {
				pd(GetIp())
				/*
					args := c.Args()
					if len(args) < 1 {
						shell.ErrorExit("Missing domain argument.\n\nExample:\n\n $ aliyundns list example.com")
					}
					List(args.Get(0))
				*/
			},
		},
		{
			Name:  "update",
			Usage: "<record_id> <rr> <value>",
			Action: func(c *cli.Context) {
				args := c.Args()
				if len(args) < 3 {
					shell.ErrorExit("Must provide 3 arguments.\n\nExample:\n\n $ aliyundns update 1234567 www 192.168.1.1")
				}
				err := Update(args.Get(0), args.Get(1), args.Get(2))
				if err != nil {
					shell.ErrorExit(err)
				}
				fmt.Println("Success")
			},
		},
		{
			Name:  "server",
			Usage: "<port>",
			Action: func(c *cli.Context) {
				args := c.Args()
				if len(args) < 1 {
					shell.ErrorExit("Must provide 2 arguments.\n\nExample:\n\n $ aliyundns server 3000")
				}
				Server(args.Get(0))
			},
		},
	}

	rc = loadRc()
	app.Run(os.Args)
}

func loadRc() (ret Rc) {
	file := os.Getenv("HOME") + "/.aliyundnsrc"

	if ok, _ := fil.IsNotExist(file); ok {
		return Rc{}
	}

	if _, err := toml.DecodeFile(file, &ret); err != nil {
		shell.ErrorExit("%s: %s\n", "Load "+file, err)
	}
	return ret
}

func List(domain string) {
	client := dns.NewClient(rc.ACCESS_KEY_ID, rc.ACCESS_KEY_SECRET)
	res, err := client.DescribeDomainRecords(&dns.DescribeDomainRecordsArgs{
		DomainName: domain,
	})
	if err != nil {
		shell.ErrorExit(err)
	}

	for _, v := range res.DomainRecords.Record {
		fmt.Println(v.RecordId, v.RR, v.Value, v.DomainName, v.Type)
	}
}

func Update(recordId, rr, value string) error {
	client := dns.NewClient(rc.ACCESS_KEY_ID, rc.ACCESS_KEY_SECRET)
	_, err := client.UpdateDomainRecord(&dns.UpdateDomainRecordArgs{
		RecordId: recordId,
		RR:       rr,
		Value:    value,
		Type:     dns.ARecord,
	})
	return err
}

func GetIp() string {
	res, _ := http.Get("http://pv.sohu.com/cityjson?ie=utf-8")
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	ip := regexp.MustCompile(`\d\d\d\.\d\d\d.\d\d\d.\d\d\d`).FindString(string(body))
	return ip
}

func Server(port string) {
	http.HandleFunc("/", homeHandler)
	fmt.Println(">> Listen on " + port)
	err := http.ListenAndServe(":"+port, nil)
	log.Fatal(err)
}

func homeHandler(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		rc.ACCESS_KEY_ID = req.FormValue("access_key_id")
		rc.ACCESS_KEY_SECRET = req.FormValue("access_key_secret")
		err := Update(req.FormValue("id"), req.FormValue("rr"), req.FormValue("value"))
		if err != nil {
			http.Error(res, err.Error(), 400)
		} else {
			fmt.Fprint(res, "Success")
		}
	}
}
