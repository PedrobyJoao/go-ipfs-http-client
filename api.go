package httpapi

import (
	"fmt"
	"io/ioutil"
	gohttp "net/http"
	"os"
	"path"
	"strings"

	iface "github.com/ipfs/interface-go-ipfs-core"
	caopts "github.com/ipfs/interface-go-ipfs-core/options"
	homedir "github.com/mitchellh/go-homedir"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

const (
	DefaultPathName = ".ipfs"
	DefaultPathRoot = "~/" + DefaultPathName
	DefaultApiFile  = "api"
	EnvDir          = "IPFS_PATH"
)

type HttpApi struct {
	url     string
	httpcli gohttp.Client

	applyGlobal func(*RequestBuilder)
}

//TODO: Return errors here
func NewLocalApi() iface.CoreAPI {
	baseDir := os.Getenv(EnvDir)
	if baseDir == "" {
		baseDir = DefaultPathRoot
	}

	return NewPathApi(baseDir)
}

func NewPathApi(p string) iface.CoreAPI {
	a := ApiAddr(p)
	if a == nil {
		return nil
	}
	return NewApi(a)
}

func ApiAddr(ipfspath string) ma.Multiaddr {
	baseDir, err := homedir.Expand(ipfspath)
	if err != nil {
		return nil
	}

	apiFile := path.Join(baseDir, DefaultApiFile)

	if _, err := os.Stat(apiFile); err != nil {
		return nil
	}

	api, err := ioutil.ReadFile(apiFile)
	if err != nil {
		return nil
	}

	maddr, err := ma.NewMultiaddr(strings.TrimSpace(string(api)))
	if err != nil {
		return nil
	}

	return maddr
}

func NewApi(a ma.Multiaddr) *HttpApi { // TODO: should be MAddr?
	c := &gohttp.Client{
		Transport: &gohttp.Transport{
			Proxy:             gohttp.ProxyFromEnvironment,
			DisableKeepAlives: true,
		},
	}

	return NewApiWithClient(a, c)
}

func NewApiWithClient(a ma.Multiaddr, c *gohttp.Client) *HttpApi {
	_, url, err := manet.DialArgs(a)
	if err != nil {
		return nil // TODO: return that error
	}

	if a, err := ma.NewMultiaddr(url); err == nil {
		_, host, err := manet.DialArgs(a)
		if err == nil {
			url = host
		}
	}

	api := &HttpApi{
		url:         url,
		httpcli:     *c,
		applyGlobal: func(*RequestBuilder) {},
	}

	// We don't support redirects.
	api.httpcli.CheckRedirect = func(_ *gohttp.Request, _ []*gohttp.Request) error {
		return fmt.Errorf("unexpected redirect")
	}

	return api
}

func (api *HttpApi) WithOptions(opts ...caopts.ApiOption) (iface.CoreAPI, error) {
	options, err := caopts.ApiOptions(opts...)
	if err != nil {
		return nil, err
	}

	subApi := *api
	subApi.applyGlobal = func(req *RequestBuilder) {
		if options.Offline {
			req.Option("offline", options.Offline)
		}
	}

	return &subApi, nil
}

func (api *HttpApi) request(command string, args ...string) *RequestBuilder {
	return &RequestBuilder{
		command: command,
		args:    args,
		shell:   api,
	}
}

func (api *HttpApi) Unixfs() iface.UnixfsAPI {
	return (*UnixfsAPI)(api)
}

func (api *HttpApi) Block() iface.BlockAPI {
	return (*BlockAPI)(api)
}

func (api *HttpApi) Dag() iface.APIDagService {
	return (*HttpDagServ)(api)
}

func (api *HttpApi) Name() iface.NameAPI {
	return (*NameAPI)(api)
}

func (api *HttpApi) Key() iface.KeyAPI {
	return (*KeyAPI)(api)
}

func (api *HttpApi) Pin() iface.PinAPI {
	return (*PinAPI)(api)
}

func (api *HttpApi) Object() iface.ObjectAPI {
	return (*ObjectAPI)(api)
}

func (api *HttpApi) Dht() iface.DhtAPI {
	return (*DhtAPI)(api)
}

func (api *HttpApi) Swarm() iface.SwarmAPI {
	return (*SwarmAPI)(api)
}

func (api *HttpApi) PubSub() iface.PubSubAPI {
	return (*PubsubAPI)(api)
}
