package persistence

import (
	"errors"
	"fmt"
	"log"
	"net/url"

	"gopkg.in/mgo.v2"
)

// Persistor persist
type Persistor struct {
	session    *mgo.Session
	url        string
	db         string
	collection string
}

// NewPersistor constructor
func NewPersistor(mongoURL string, collection string) (p *Persistor, err error) {
	log.Println("creating new persistor with db:", mongoURL, "and collection:", collection)
	parsedURL, err := url.Parse(mongoURL)
	if err != nil {
		return nil, err
	}
	if parsedURL.Scheme != "mongodb" {
		return nil, fmt.Errorf("missing scheme mongo:// in %q", mongoURL)
	}
	if len(parsedURL.Path) < 2 {
		return nil, errors.New("invalid mongoURL missing db should be mongodb://server:port/db")
	}
	session, err := mgo.Dial(mongoURL)
	if err != nil {
		return nil, err
	}
	p = &Persistor{
		session:    session,
		url:        mongoURL,
		db:         parsedURL.Path[1:],
		collection: collection,
	}
	return p, nil
}

func (p *Persistor) GetCollection() *mgo.Collection {
	return p.session.DB(p.db).C(p.collection)
}
func (p *Persistor) GetCollectionName() string {
	return p.collection
}
func (p *Persistor) GetURL() string {
	return p.url
}