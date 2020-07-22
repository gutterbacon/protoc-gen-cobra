// Code generated by protoc-gen-cobra. DO NOT EDIT.

package pb

import (
	context "context"
	tls "crypto/tls"
	x509 "crypto/x509"
	fmt "fmt"
	iocodec "github.com/NathanBaulch/protoc-gen-cobra/iocodec"
	proto "github.com/golang/protobuf/proto"
	cobra "github.com/spf13/cobra"
	pflag "github.com/spf13/pflag"
	oauth2 "golang.org/x/oauth2"
	grpc "google.golang.org/grpc"
	credentials "google.golang.org/grpc/credentials"
	oauth "google.golang.org/grpc/credentials/oauth"
	ioutil "io/ioutil"
	net "net"
	os "os"
	filepath "path/filepath"
	strings "strings"
	time "time"
)

var TypesClientDefaultConfig = &_TypesClientConfig{
	ServerAddr:     "localhost:8080",
	ResponseFormat: "json",
	Timeout:        10 * time.Second,
	AuthTokenType:  "Bearer",
}

type _TypesClientConfig struct {
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

func (o *_TypesClientConfig) addFlags(fs *pflag.FlagSet) {
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

func TypesClientCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "types",
		Short: "Types service client",
		Long:  "",
	}
	TypesClientDefaultConfig.addFlags(cmd.PersistentFlags())
	cmd.AddCommand(
		_TypesEchoCommand(),
	)
	return cmd
}

func _TypesDial(ctx context.Context) (*grpc.ClientConn, TypesClient, error) {
	cfg := TypesClientDefaultConfig
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
			TokenType:   cfg.AuthTokenType,
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
	return conn, NewTypesClient(conn), nil
}

type _TypesRoundTripFunc func(cli TypesClient, in iocodec.Decoder, out iocodec.Encoder) error

func _TypesRoundTrip(ctx context.Context, fn _TypesRoundTripFunc) error {
	cfg := TypesClientDefaultConfig
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
	conn, client, err := _TypesDial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return fn(client, dm.NewDecoder(r), em.NewEncoder(os.Stdout))
}

func _TypesEchoCommand() *cobra.Command {
	req := &Sound{}

	cmd := &cobra.Command{
		Use:   "echo",
		Short: "Echo RPC client",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return _TypesRoundTrip(cmd.Context(), func(cli TypesClient, in iocodec.Decoder, out iocodec.Encoder) error {
				v := &Sound{}

				if err := in.Decode(v); err != nil {
					return err
				}
				proto.Merge(v, req)

				res, err := cli.Echo(cmd.Context(), v)

				if err != nil {
					return err
				}

				return out.Encode(res)

			})
		},
	}

	cmd.PersistentFlags().BoolSliceVar(&req.ListBool, "listbool", nil, "")
	cmd.PersistentFlags().BoolVar(&req.Bool, "bool", false, "")
	cmd.PersistentFlags().BytesBase64Var(&req.Bytes, "bytes", nil, "")
	cmd.PersistentFlags().Float32SliceVar(&req.ListFloat, "listfloat", nil, "")
	cmd.PersistentFlags().Float32Var(&req.Float, "float", 0, "")
	cmd.PersistentFlags().Float64SliceVar(&req.ListDouble, "listdouble", nil, "")
	cmd.PersistentFlags().Float64Var(&req.Double, "double", 0, "")
	cmd.PersistentFlags().Int32SliceVar(&req.ListInt32, "listint32", nil, "")
	cmd.PersistentFlags().Int32SliceVar(&req.ListSfixed32, "listsfixed32", nil, "")
	cmd.PersistentFlags().Int32SliceVar(&req.ListSint32, "listsint32", nil, "")
	cmd.PersistentFlags().Int32Var(&req.Int32, "int32", 0, "")
	cmd.PersistentFlags().Int32Var(&req.Sfixed32, "sfixed32", 0, "")
	cmd.PersistentFlags().Int32Var(&req.Sint32, "sint32", 0, "")
	cmd.PersistentFlags().Int64SliceVar(&req.ListInt64, "listint64", nil, "")
	cmd.PersistentFlags().Int64SliceVar(&req.ListSfixed64, "listsfixed64", nil, "")
	cmd.PersistentFlags().Int64SliceVar(&req.ListSint64, "listsint64", nil, "")
	cmd.PersistentFlags().Int64Var(&req.Int64, "int64", 0, "")
	cmd.PersistentFlags().Int64Var(&req.Sfixed64, "sfixed64", 0, "")
	cmd.PersistentFlags().Int64Var(&req.Sint64, "sint64", 0, "")
	cmd.PersistentFlags().StringSliceVar(&req.ListString, "liststring", nil, "")
	cmd.PersistentFlags().StringToInt64Var(&req.MapStringInt64, "mapstringint64", nil, "")
	cmd.PersistentFlags().StringToStringVar(&req.MapStringString, "mapstringstring", nil, "")
	cmd.PersistentFlags().StringVar(&req.String_, "string_", "", "")
	cmd.PersistentFlags().Uint32Var(&req.Fixed32, "fixed32", 0, "")
	cmd.PersistentFlags().Uint32Var(&req.Uint32, "uint32", 0, "")
	cmd.PersistentFlags().Uint64Var(&req.Fixed64, "fixed64", 0, "")
	cmd.PersistentFlags().Uint64Var(&req.Uint64, "uint64", 0, "")

	return cmd
}
