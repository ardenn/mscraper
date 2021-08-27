package main

import (
	"bytes"
	"io"
	neturl "net/url"
	"reflect"
	"sync"
	"testing"

	"golang.org/x/net/html"
)

type fakeCrawler struct {
	store   *Store
	host    string
	scheme  string
	verbose bool
}

func (c *fakeCrawler) Fetch(url string) (urls []string, err error) {
	c.store.write(url)
	return []string{"/home", "/faq"}, nil
}

func (c *fakeCrawler) Verbose() bool {
	return c.verbose
}

func TestStore_write(t *testing.T) {
	type fields struct {
		mutex        *sync.RWMutex
		VisitedLinks map[string]struct{}
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test-can-write-store",
			args: args{
				key: "key",
			},
			fields: fields{
				mutex:        &sync.RWMutex{},
				VisitedLinks: make(map[string]struct{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{
				mutex:        tt.fields.mutex,
				VisitedLinks: tt.fields.VisitedLinks,
			}
			s.write(tt.args.key)
		})
	}
}

func TestStore_read(t *testing.T) {
	type fields struct {
		mutex        *sync.RWMutex
		VisitedLinks map[string]struct{}
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "test-can-read-store-with-key",
			args: args{
				key: "key",
			},
			fields: fields{
				mutex:        &sync.RWMutex{},
				VisitedLinks: map[string]struct{}{"key": {}},
			},
			want: true,
		},
		{
			name: "test-can-read-store-without-key",
			args: args{
				key: "key",
			},
			fields: fields{
				mutex:        &sync.RWMutex{},
				VisitedLinks: make(map[string]struct{}),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{
				mutex:        tt.fields.mutex,
				VisitedLinks: tt.fields.VisitedLinks,
			}
			if got := s.read(tt.args.key); got != tt.want {
				t.Errorf("Store.read() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_crawl(t *testing.T) {
	link, _ := neturl.Parse("https://example.com")
	type args struct {
		url   neturl.URL
		depth int
		store *Store
		c     HTTPFetcher
		wg    *sync.WaitGroup
		want  int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test-can-crawl-success-depth-0",
			args: args{
				url:   *link,
				depth: 0,
				store: &Store{},
				c:     &fakeCrawler{},
				wg:    &sync.WaitGroup{},
				want:  1,
			},
		},
		{
			name: "test-can-crawl-success-depth-1",
			args: args{
				url:   *link,
				depth: 1,
				store: &Store{},
				c:     &fakeCrawler{},
				wg:    &sync.WaitGroup{},
				want:  3,
			},
		},
	}
	for _, tt := range tests {
		store := &Store{mutex: &sync.RWMutex{}, VisitedLinks: make(map[string]struct{})}
		tt.args.store = store
		tt.args.c = &fakeCrawler{host: link.Host, scheme: link.Scheme, store: store, verbose: false}
		t.Run(tt.name, func(t *testing.T) {
			tt.args.wg.Add(1)
			crawl(tt.args.url, tt.args.depth, tt.args.store, tt.args.c, tt.args.wg)
			tt.args.wg.Wait()
			if len(store.VisitedLinks) != tt.args.want {
				t.Errorf("len(VisitedLinks) = %d, wanted %d", len(store.VisitedLinks), tt.args.want)
			}
		})
	}
}

func TestCrawler_Fetch(t *testing.T) {
	type fields struct {
		store   *Store
		host    string
		scheme  string
		verbose bool
	}
	type args struct {
		url string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantUrls []string
		wantErr  bool
	}{
		{
			name: "test-can-fetch-success",
			fields: fields{
				store:   &Store{VisitedLinks: make(map[string]struct{}), mutex: &sync.RWMutex{}},
				host:    "example.com",
				scheme:  "https",
				verbose: false,
			},
			args: args{
				url: "https://example.com",
			},
			wantUrls: []string{"/home", "/faq"},
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &fakeCrawler{
				store:   tt.fields.store,
				host:    tt.fields.host,
				scheme:  tt.fields.scheme,
				verbose: tt.fields.verbose,
			}
			gotUrls, err := c.Fetch(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("Crawler.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotUrls, tt.wantUrls) {
				t.Errorf("Crawler.Fetch() = %v, want %v", gotUrls, tt.wantUrls)
			}
		})
	}
}

func TestCrawler_processPage(t *testing.T) {
	type fields struct {
		store   *Store
		host    string
		scheme  string
		verbose bool
	}
	type args struct {
		page io.Reader
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantUrls []string
		wantErr  bool
	}{
		{
			name: "test-can-processpage-success",
			fields: fields{
				store:   &Store{VisitedLinks: make(map[string]struct{}), mutex: &sync.RWMutex{}},
				host:    "example.com",
				scheme:  "https",
				verbose: false,
			},
			args: args{
				page: bytes.NewReader([]byte(`<p>Links:</p><ul><li><a href="foo">Foo</a><li><a href="/bar/baz">BarBaz</a></ul>`)),
			},
			wantUrls: []string{"foo", "/bar/baz"},
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawler{
				store:   tt.fields.store,
				host:    tt.fields.host,
				scheme:  tt.fields.scheme,
				verbose: tt.fields.verbose,
			}
			gotUrls, err := c.processPage(tt.args.page)
			if (err != nil) != tt.wantErr {
				t.Errorf("Crawler.processPage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotUrls, tt.wantUrls) {
				t.Errorf("Crawler.processPage() = %v, want %v", gotUrls, tt.wantUrls)
			}
		})
	}
}

func TestCrawler_Verbose(t *testing.T) {
	type fields struct {
		store   *Store
		host    string
		scheme  string
		verbose bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "test-verbose-true",
			fields: fields{
				store:   &Store{VisitedLinks: make(map[string]struct{}), mutex: &sync.RWMutex{}},
				host:    "example.com",
				scheme:  "https",
				verbose: true,
			},
			want: true,
		},
		{
			name: "test-verbose-false",
			fields: fields{
				store:   &Store{VisitedLinks: make(map[string]struct{}), mutex: &sync.RWMutex{}},
				host:    "example.com",
				scheme:  "https",
				verbose: false,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawler{
				store:   tt.fields.store,
				host:    tt.fields.host,
				scheme:  tt.fields.scheme,
				verbose: tt.fields.verbose,
			}
			if got := c.Verbose(); got != tt.want {
				t.Errorf("Crawler.Verbose() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCrawler_processLink(t *testing.T) {
	type fields struct {
		store   *Store
		host    string
		scheme  string
		verbose bool
	}
	type args struct {
		link string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
		want1  string
	}{
		{
			name: "test-process-external-link",
			fields: fields{
				store:   &Store{VisitedLinks: make(map[string]struct{}), mutex: &sync.RWMutex{}},
				host:    "example.com",
				scheme:  "https",
				verbose: false,
			},
			args: args{
				link: "https://examplee.com/home",
			},
			want:  false,
			want1: "",
		},
		{
			name: "test-process-relative-link",
			fields: fields{
				store:   &Store{VisitedLinks: make(map[string]struct{}), mutex: &sync.RWMutex{}},
				host:    "example.com",
				scheme:  "https",
				verbose: false,
			},
			args: args{
				link: "../home",
			},
			want:  true,
			want1: "../home",
		},
		{
			name: "test-process-internal-link",
			fields: fields{
				store:   &Store{VisitedLinks: make(map[string]struct{}), mutex: &sync.RWMutex{}},
				host:    "example.com",
				scheme:  "https",
				verbose: false,
			},
			args: args{
				link: "https://example.com/home",
			},
			want:  true,
			want1: "https://example.com/home",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawler{
				store:   tt.fields.store,
				host:    tt.fields.host,
				scheme:  tt.fields.scheme,
				verbose: tt.fields.verbose,
			}
			got, got1 := c.processLink(tt.args.link)
			if got != tt.want {
				t.Errorf("Crawler.processLink() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Crawler.processLink() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestCrawler_findLinks(t *testing.T) {
	node, _ := html.Parse(bytes.NewReader([]byte(`<p>Links:</p><ul><li><a href="foo">Foo</a><li><a href="/bar/baz">BarBaz</a></ul>`)))
	type fields struct {
		store   *Store
		host    string
		scheme  string
		verbose bool
	}
	type args struct {
		node       *html.Node
		foundLinks []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "test-findlinks-success",
			fields: fields{
				store:   &Store{VisitedLinks: make(map[string]struct{}), mutex: &sync.RWMutex{}},
				host:    "example.com",
				scheme:  "https",
				verbose: false,
			},
			args: args{
				node:       node,
				foundLinks: make([]string, 0),
			},
			want: []string{"foo", "/bar/baz"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawler{
				store:   tt.fields.store,
				host:    tt.fields.host,
				scheme:  tt.fields.scheme,
				verbose: tt.fields.verbose,
			}
			if got := c.findLinks(tt.args.node, tt.args.foundLinks); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Crawler.findLinks() = %v, want %v", got, tt.want)
			}
		})
	}
}
