package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/cli/browser"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const (
	defaultClientID     = "llm-operator"
	defaultClientSecret = "ZXhhbXBsZS1hcHAtc2VjcmV0"
	defaultRedirectURI  = "http://127.0.0.1:5555/callback"
	defaultIssuerURL    = "http://kong-kong-proxy.kong/dex"
	defaultNodeIP       = "127.0.0.1"
)

type client struct {
	clientID     string
	clientSecret string
	redirectURI  string

	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	listener net.Listener
}

func main() {
	if err := cmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}

func cmd() *cobra.Command {
	var (
		cli       client
		issuerURL string
		nodeIP    string
	)
	cmd := cobra.Command{
		Use: "login",
		RunE: func(cmd *cobra.Command, args []string) error {
			ru, err := url.Parse(cli.redirectURI)
			if err != nil {
				return fmt.Errorf("parse redirect-uri: %v", err)
			}
			iu, err := url.Parse(issuerURL)
			if err != nil {
				return fmt.Errorf("parse issuer-uri: %v", err)
			}

			dialer := &net.Dialer{}
			http.DefaultTransport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				if addr == fmt.Sprintf("%s:80", iu.Host) {
					addr = fmt.Sprintf("%s:80", nodeIP)
				}
				return dialer.DialContext(ctx, network, addr)
			}

			ctx := oidc.ClientContext(context.Background(), http.DefaultClient)
			provider, err := oidc.NewProvider(ctx, issuerURL)
			if err != nil {
				return err
			}
			cli.provider = provider
			cli.verifier = provider.Verifier(&oidc.Config{ClientID: cli.clientID})

			iu.Host = nodeIP
			iu.Path = path.Join(iu.Path, "auth")
			q := iu.Query()
			q.Add("client_id", cli.clientID)
			q.Add("redirect_uri", cli.redirectURI)
			q.Add("response_type", "code")
			q.Add("scope", "openid profile email")
			iu.RawQuery = q.Encode()
			fmt.Println("Open browser...")
			if err := browser.OpenURL(iu.String()); err != nil {
				return err
			}

			l, err := net.Listen("tcp", ru.Host)
			if err != nil {
				return err
			}
			cli.listener = l
			http.HandleFunc(ru.Path, cli.handleCallback)
			http.Serve(l, nil)
			return nil
		},
	}
	cmd.Flags().StringVar(&cli.clientID, "client-id", defaultClientID, "OAuth2 client ID of this application.")
	cmd.Flags().StringVar(&cli.clientSecret, "client-secret", defaultClientSecret, "OAuth2 client secret of this application.")
	cmd.Flags().StringVar(&cli.redirectURI, "redirect-uri", defaultRedirectURI, "Callback URL for OAuth2 responses.")
	cmd.Flags().StringVar(&issuerURL, "issuer", defaultIssuerURL, "URL of the OpenID Connect issuer.")
	cmd.Flags().StringVar(&nodeIP, "node-ip", defaultNodeIP, "IP address of the k8s node.")
	return &cmd
}

func (c *client) stop() {
	go func() {
		time.Sleep(time.Second)
		c.listener.Close()
	}()
}

func (c *client) handleCallback(w http.ResponseWriter, r *http.Request) {
	var (
		err   error
		token *oauth2.Token
	)

	ctx := oidc.ClientContext(r.Context(), http.DefaultClient)
	oauth2Config := &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     c.provider.Endpoint(),
		RedirectURL:  c.redirectURI,
	}

	if r.Method == http.MethodGet {
		if errMsg := r.FormValue("error"); errMsg != "" {
			http.Error(w, fmt.Sprintf("%s: %s", errMsg, r.FormValue("error_description")), http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		if code == "" {
			http.Error(w, fmt.Sprintf("no code in request: %q", r.Form), http.StatusBadRequest)
			return
		}
		token, err = oauth2Config.Exchange(ctx, code)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get token: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, fmt.Sprintf("method not implemented: %s", r.Method), http.StatusBadRequest)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token in token response", http.StatusInternalServerError)
		return
	}
	if _, err := c.verifier.Verify(r.Context(), rawIDToken); err != nil {
		http.Error(w, fmt.Sprintf("failed to verify ID token: %v", err), http.StatusInternalServerError)
		return
	}

	accessToken, ok := token.Extra("access_token").(string)
	if !ok {
		http.Error(w, "no access_token in token response", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Successfully logged in"))
	fmt.Println("token:", accessToken)
	c.stop()
}
