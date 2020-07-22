package main

import (
	"fmt"
	"sort"
	"strings"
	"text/template"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func genFile(gen *protogen.Plugin, file *protogen.File) error {
	if len(file.Services) == 0 {
		return nil
	}

	g := gen.NewGeneratedFile(file.GeneratedFilenamePrefix+".cobra.pb.go", file.GoImportPath)
	g.P("// Code generated by protoc-gen-cobra. DO NOT EDIT.")
	g.P()
	g.P("package ", file.GoPackageName)
	g.P()

	for _, srv := range file.Services {
		if err := genService(g, srv); err != nil {
			return err
		}
	}

	return nil
}

var (
	serviceTemplateCode = `
var {{.GoName}}ClientDefaultConfig = &_{{.GoName}}ClientConfig{
	ServerAddr: "localhost:8080",
	ResponseFormat: "json",
	Timeout: 10 * time.Second,
	AuthTokenType: "Bearer",
}

type _{{.GoName}}ClientConfig struct {
	ServerAddr         string
	RequestFile        string
	Stdin              bool
	ResponseFormat     string
	Timeout            time.Duration
	TLS                bool
	ServerName         string
	InsecureSkipVerify bool
	CACertFile         string
	CertFile           string
	KeyFile            string
	AuthToken          string
	AuthTokenType      string
	JWTKey             string
	JWTKeyFile         string
}

func (o *_{{.GoName}}ClientConfig) addFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.ServerAddr, "server-addr", "s", o.ServerAddr, "server address in form of host:port")
	fs.StringVarP(&o.RequestFile, "request-file", "f", o.RequestFile, "client request file (must be json, yaml, or xml); use \"-\" for stdin + json")
	fs.BoolVar(&o.Stdin, "stdin", o.Stdin, "read client request from STDIN; alternative for '-f -'")
	fs.StringVarP(&o.ResponseFormat, "response-format", "o", o.ResponseFormat, "response format (json, prettyjson, xml, prettyxml, or yaml)")
	fs.DurationVar(&o.Timeout, "timeout", o.Timeout, "client connection timeout")
	fs.BoolVar(&o.TLS, "tls", o.TLS, "enable tls")
	fs.StringVar(&o.ServerName, "tls-server-name", o.ServerName, "tls server name override")
	fs.BoolVar(&o.InsecureSkipVerify, "tls-insecure-skip-verify", o.InsecureSkipVerify, "INSECURE: skip tls checks")
	fs.StringVar(&o.CACertFile, "tls-ca-cert-file", o.CACertFile, "ca certificate file")
	fs.StringVar(&o.CertFile, "tls-cert-file", o.CertFile, "client certificate file")
	fs.StringVar(&o.KeyFile, "tls-key-file", o.KeyFile, "client key file")
	fs.StringVar(&o.AuthToken, "auth-token", o.AuthToken, "authorization token")
	fs.StringVar(&o.AuthTokenType, "auth-token-type", o.AuthTokenType, "authorization token type")
	fs.StringVar(&o.JWTKey, "jwt-key", o.JWTKey, "jwt key")
	fs.StringVar(&o.JWTKeyFile, "jwt-key-file", o.JWTKeyFile, "jwt key file")
}

func {{.GoName}}ClientCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "{{.GoName | toLower}}",
		Short: "{{.GoName}} service client",
		Long: "{{.Comments.Leading | cleanComments}}",
	}
	{{.GoName}}ClientDefaultConfig.addFlags(cmd.PersistentFlags())
	cmd.AddCommand({{range .Methods}}
		_{{$.GoName}}{{.GoName}}Command(),{{end}}
	)
	return cmd
}

func _{{.GoName}}Dial(ctx context.Context) (*grpc.ClientConn, {{.GoName}}Client, error) {
	cfg := {{.GoName}}ClientDefaultConfig
	opts := []grpc.DialOption{grpc.WithBlock()}
	if cfg.TLS {
		tlsConfig := &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}
		if cfg.CACertFile != "" {
			caCert, err := ioutil.ReadFile(cfg.CACertFile)
			if err != nil {
				return nil, nil, fmt.Errorf("ca cert: %v", err)
			}
			certPool := x509.NewCertPool()
			certPool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = certPool
		}
		if cfg.CertFile != "" {
			if cfg.KeyFile == "" {
				return nil, nil, fmt.Errorf("missing key file")
			}
			pair, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
			if err != nil {
				return nil, nil, fmt.Errorf("cert/key: %v", err)
			}
			tlsConfig.Certificates = []tls.Certificate{pair}
		}
		if cfg.ServerName != "" {
			tlsConfig.ServerName = cfg.ServerName
		} else {
			addr, _, _ := net.SplitHostPort(cfg.ServerAddr)
			tlsConfig.ServerName = addr
		}
		cred := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(cred))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	if cfg.AuthToken != "" {
		cred := oauth.NewOauthAccess(&oauth2.Token{
			AccessToken: cfg.AuthToken,
			TokenType: cfg.AuthTokenType,
		})
		opts = append(opts, grpc.WithPerRPCCredentials(cred))
	}
	if cfg.JWTKey != "" {
		cred, err := oauth.NewJWTAccessFromKey([]byte(cfg.JWTKey))
		if err != nil {
			return nil, nil, fmt.Errorf("jwt key: %v", err)
		}
		opts = append(opts, grpc.WithPerRPCCredentials(cred))
	}
	if cfg.JWTKeyFile != "" {
		cred, err := oauth.NewJWTAccessFromFile(cfg.JWTKeyFile)
		if err != nil {
			return nil, nil, fmt.Errorf("jwt key file: %v", err)
		}
		opts = append(opts, grpc.WithPerRPCCredentials(cred))
	}
	if cfg.Timeout > 0 {
		var done context.CancelFunc
		ctx, done = context.WithTimeout(ctx, cfg.Timeout)
		defer done()
	}
	conn, err := grpc.DialContext(ctx, cfg.ServerAddr, opts...)
	if err != nil {
		return nil, nil, err
	}
	return conn, New{{.GoName}}Client(conn), nil
}

type _{{.GoName}}RoundTripFunc func(cli {{.GoName}}Client, in iocodec.Decoder, out iocodec.Encoder) error

func _{{.GoName}}RoundTrip(ctx context.Context, fn _{{.GoName}}RoundTripFunc) error {
	cfg := {{.GoName}}ClientDefaultConfig
	var dm iocodec.DecoderMaker
	r := os.Stdin
	if cfg.Stdin || cfg.RequestFile == "-" {
		dm = iocodec.DefaultDecoders["json"]
	} else if cfg.RequestFile != "" {
		f, err := os.Open(cfg.RequestFile)
		if err != nil {
			return fmt.Errorf("request file: %v", err)
		}
		defer f.Close()
		if ext := strings.TrimLeft(filepath.Ext(cfg.RequestFile), "."); ext != "" {
			dm = iocodec.DefaultDecoders[ext]
		}
		if dm == nil {
			dm = iocodec.DefaultDecoders["json"]
		}
		r = f
	} else {
		dm = iocodec.DefaultDecoders["noop"]
	}
	var em iocodec.EncoderMaker
	if cfg.ResponseFormat == "" {
		em = iocodec.DefaultEncoders["json"]
	} else if em = iocodec.DefaultEncoders[cfg.ResponseFormat]; em == nil {
		return fmt.Errorf("invalid response format: %q", cfg.ResponseFormat)
	}
	conn, client, err := _{{.GoName}}Dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return fn(client, dm.NewDecoder(r), em.NewEncoder(os.Stdout))
}
`
	serviceTemplate = template.Must(template.New("service").
		Funcs(template.FuncMap{"toLower": strings.ToLower, "cleanComments": cleanComments}).
		Parse(serviceTemplateCode))
	serviceImports = []protogen.GoImportPath{
		"context",
		"crypto/tls",
		"crypto/x509",
		"fmt",
		"io/ioutil",
		"net",
		"os",
		"path/filepath",
		"strings",
		"time",
		"github.com/NathanBaulch/protoc-gen-cobra/iocodec",
		"github.com/spf13/cobra",
		"github.com/spf13/pflag",
		"golang.org/x/oauth2",
		"google.golang.org/grpc",
		"google.golang.org/grpc/credentials",
		"google.golang.org/grpc/credentials/oauth",
	}
)

func genService(g *protogen.GeneratedFile, service *protogen.Service) error {
	for _, imp := range serviceImports {
		g.QualifiedGoIdent(protogen.GoIdent{GoImportPath: imp})
	}
	if err := serviceTemplate.Execute(g, service); err != nil {
		return err
	}

	for _, mth := range service.Methods {
		if err := genMethod(g, mth); err != nil {
			return err
		}
	}

	return nil
}

var (
	methodTemplateCode = `
func _{{.Parent.GoName}}{{.GoName}}Command() *cobra.Command {
	req := {{.InputInitializerCode}}

	cmd := &cobra.Command{
		Use: "{{.GoName | toLower}}",
		Short: "{{.GoName}} RPC client",
		Long: "{{.Comments.Leading | cleanComments}}",
		RunE: func(cmd *cobra.Command, args []string) error {
			return _{{.Parent.GoName}}RoundTrip(cmd.Context(), func(cli {{.Parent.GoName}}Client, in iocodec.Decoder, out iocodec.Encoder) error {
				v := &{{.Input.GoIdent.GoName}}{}
	{{if .Desc.IsStreamingClient}}
				stm, err := cli.{{.GoName}}(cmd.Context())
				if err != nil {
					return err
				}
				for {
					if err := in.Decode(v); err != nil {
						if err == io.EOF {
							_ = stm.CloseSend()
							break
						}
						return err
					}
					proto.Merge(v, req)
					if err = stm.Send(v); err != nil {
						return err
					}
				}
	{{else}}
				if err := in.Decode(v); err != nil {
					return err
				}
				proto.Merge(v, req)
		{{if .Desc.IsStreamingServer}}
				stm, err := cli.{{.GoName}}(cmd.Context(), v)
		{{else}}
				res, err := cli.{{.GoName}}(cmd.Context(), v)
		{{end}}
				if err != nil {
					return err
				}
	{{end}}
	{{if .Desc.IsStreamingServer}}
				for {
					res, err := stm.Recv()
					if err != nil {
						if err == io.EOF {
							break
						}
						return err
					}
					if err = out.Encode(res); err != nil {
						return err
					}
				}
				return nil
	{{else}}
		{{if .Desc.IsStreamingClient}}
				res, err := stm.CloseAndRecv()
				if err != nil {
					return err
				}
		{{end}}
				return out.Encode(res)
	{{end}}
			})
		},
	}

	{{.InputFieldFlagCode}}

	return cmd
}
`
	methodTemplate = template.Must(template.New("method").
		Funcs(template.FuncMap{"toLower": strings.ToLower, "cleanComments": cleanComments}).
		Parse(methodTemplateCode))
	methodImports = []protogen.GoImportPath{
		"github.com/golang/protobuf/proto",
		"github.com/NathanBaulch/protoc-gen-cobra/iocodec",
		"github.com/spf13/cobra",
	}
)

func genMethod(g *protogen.GeneratedFile, method *protogen.Method) error {
	for _, imp := range methodImports {
		g.QualifiedGoIdent(protogen.GoIdent{GoImportPath: imp})
	}
	if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
		g.QualifiedGoIdent(protogen.GoIdent{GoImportPath: "io"})
	}

	initCode, flagCode := walkFields(g, method.Input, nil)
	data := struct {
		*protogen.Method
		InputInitializerCode string
		InputFieldFlagCode   string
	}{method, initCode, flagCode}
	return methodTemplate.Execute(g, data)
}

func walkFields(g *protogen.GeneratedFile, message *protogen.Message, path []string) (string, string) {
	var initLines []string
	flagLines := make([]string, 0, len(message.Fields))

	for _, fld := range message.Fields {
		var flagLine string
		path := append(path, fld.GoName)
		goPath := strings.Join(path, ".")
		flagName := strings.ToLower(strings.Join(path, "-"))
		comment := cleanComments(fld.Comments.Leading)

		switch fld.Desc.Kind() {
		case protoreflect.BoolKind:
			if fld.Desc.IsList() {
				flagLine = fmt.Sprintf("BoolSliceVar(&req.%s, %q, nil, %q)", goPath, flagName, comment)
			} else {
				flagLine = fmt.Sprintf("BoolVar(&req.%s, %q, false, %q)", goPath, flagName, comment)
			}
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			if fld.Desc.IsList() {
				flagLine = fmt.Sprintf("Int32SliceVar(&req.%s, %q, nil, %q)", goPath, flagName, comment)
			} else {
				flagLine = fmt.Sprintf("Int32Var(&req.%s, %q, 0, %q)", goPath, flagName, comment)
			}
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			if fld.Desc.IsList() {
				// uint32 list not supported
			} else {
				flagLine = fmt.Sprintf("Uint32Var(&req.%s, %q, 0, %q)", goPath, flagName, comment)
			}
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			if fld.Desc.IsList() {
				flagLine = fmt.Sprintf("Int64SliceVar(&req.%s, %q, nil, %q)", goPath, flagName, comment)
			} else {
				flagLine = fmt.Sprintf("Int64Var(&req.%s, %q, 0, %q)", goPath, flagName, comment)
			}
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			if fld.Desc.IsList() {
				// uint64 list not supported
			} else {
				flagLine = fmt.Sprintf("Uint64Var(&req.%s, %q, 0, %q)", goPath, flagName, comment)
			}
		case protoreflect.FloatKind:
			if fld.Desc.IsList() {
				flagLine = fmt.Sprintf("Float32SliceVar(&req.%s, %q, nil, %q)", goPath, flagName, comment)
			} else {
				flagLine = fmt.Sprintf("Float32Var(&req.%s, %q, 0, %q)", goPath, flagName, comment)
			}
		case protoreflect.DoubleKind:
			if fld.Desc.IsList() {
				flagLine = fmt.Sprintf("Float64SliceVar(&req.%s, %q, nil, %q)", goPath, flagName, comment)
			} else {
				flagLine = fmt.Sprintf("Float64Var(&req.%s, %q, 0, %q)", goPath, flagName, comment)
			}
		case protoreflect.StringKind:
			if fld.Desc.IsList() {
				flagLine = fmt.Sprintf("StringSliceVar(&req.%s, %q, nil, %q)", goPath, flagName, comment)
			} else {
				flagLine = fmt.Sprintf("StringVar(&req.%s, %q, \"\", %q)", goPath, flagName, comment)
			}
		case protoreflect.BytesKind:
			if fld.Desc.IsList() {
				// bytes list not supported
			} else {
				flagLine = fmt.Sprintf("BytesBase64Var(&req.%s, %q, nil, %q)", goPath, flagName, comment)
			}
		case protoreflect.EnumKind:
			// enum not supported
		case protoreflect.MessageKind, protoreflect.GroupKind:
			if fld.Desc.ContainingOneof() != nil {
				// oneof not supported
			} else if fld.Desc.IsList() {
				// message list not supported
			} else if fld.Desc.IsMap() {
				// TODO: expand map support
				if fld.Desc.MapKey().Kind() == protoreflect.StringKind {
					switch fld.Desc.MapValue().Kind() {
					case protoreflect.StringKind:
						flagLine = fmt.Sprintf("StringToStringVar(&req.%s, %q, nil, %q)", goPath, flagName, comment)
					case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
						flagLine = fmt.Sprintf("StringToInt64Var(&req.%s, %q, nil, %q)", goPath, flagName, comment)
					}
				}
			} else {
				i, f := walkFields(g, fld.Message, path)
				if i != "" {
					initLines = append(initLines, fld.GoName+": "+i+",")
				}
				if f != "" {
					flagLines = append(flagLines, f)
				}
			}
		}

		if flagLine != "" {
			flagLines = append(flagLines, "cmd.PersistentFlags()."+flagLine)
		}
	}

	initCode := ""
	if len(initLines) > 0 {
		sort.Strings(initLines)
		initCode = fmt.Sprintf("\n%s\n", strings.Join(initLines, "\n"))
	}
	sort.Strings(flagLines)
	return fmt.Sprintf("&%s{%s}", g.QualifiedGoIdent(message.GoIdent), initCode), strings.Join(flagLines, "\n")
}

func cleanComments(comments protogen.Comments) string {
	return strings.TrimSpace(string(comments))
}
