package news

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
)

type Publisher struct {
	Config 		PublisherConfig
	mutex  		sync.RWMutex
	cats		[][]string
	subs 		[]chan Article
	cancel    	context.CancelFunc
	Stopped		bool
	Archive		[]Article
}

type PublisherConfig struct {
	Backupfile 	string
	Publishfile string
	Jsonformat  bool
}

// func NewPublisher() creates a new instance of a Publisher
func NewPublisher(config PublisherConfig) (*Publisher) {
	log.Println("publisher new publisher")
	p := &Publisher{}

	p.subs = []chan Article{} 	// slice of channels, the slice index serves as the subscriber ID
	p.cats = [][]string{}		// slice of slice of categories, the slice index serves as the subscriber ID
	p.Stopped = false

	log.Println(len(p.subs), len(p.cats))
	// if not specified, set backupfile to a default location
	if config.Backupfile == "" {
		config.Backupfile = "./backup.tmp"
	}

	p.Config = config

	return p
}

// func Stop stops the publisher 
func (p *Publisher) Stop() error {
	log.Println("publisher stop")
	p.mutex.Lock()
	
	if !p.Stopped {
		p.Stopped = true
		for _, ch := range p.subs {
			close(ch)
		}
	}
	p.mutex.Unlock()
	
	err  := p.Save()
	return err
}

// func Save saves the state of the publisher to the configured saving location 
func (p *Publisher) Save() error {
	log.Println("publisher save")

	p.mutex.RLock()
	defer p.mutex.RUnlock()
	data, err := json.Marshal(p)

	if err != nil {
		return err
	}

	err = os.WriteFile(p.Config.Backupfile, data, 0644);
	return err
}

// func AddSource() adds a news source to the publisher, distributes its articles
func (p *Publisher) Dispatch(s Source) {
	log.Println("publisher distribute source")

	go func() {
		// start listening to articles that are published by the source
		log.Println("publisher distribute source start listening")
		for a := range s.GetSourceChannel() {

			// add the received article to the archive 
			p.mutex.Lock()
			a.Id = len(p.Archive) + 1
			p.Archive = append(p.Archive, a)
			p.mutex.Unlock()

			lc := strings.ToLower(a.Category)

			for i, topics := range p.cats {
				for _, topic := range topics {
					if topic == lc {
						p.subs[i] <- a
					}
				}
			}
				
		}
	}()
}

// func Subscribe() adds a subscriber to a publisher. The subscriber has to
// provide the topic to which it is subscribing and its unique identifier
func (p *Publisher) Subscribe(cats []string) (<-chan Article, error) {
	log.Println("publisher subscribe")
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ch := make(chan Article)
	id := len(p.subs)
	log.Println(id)
	
	c := []string{}
	for _, cat := range cats {
		c = append(c, strings.ToLower(cat))
	} 

	p.cats = append(p.cats, c)
	p.subs = append(p.subs, ch)
	
	return ch, nil
}

// func Articles returns the articles from the archive as specified by the IDs
func (p *Publisher) Articles (ids ...int) []Article {
	log.Printf("publisher articles")

	p.mutex.RLock()
	defer p.mutex.RUnlock()
	a := []Article{}

	for _, id := range ids {
		log.Printf("publisher articles id %d", id)
		a = append(a, p.Archive[id])
	}
	return a
}

// func Clear clears the publisher archive
func (p *Publisher) Clear () {
	log.Printf("publisher clear")
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.Archive = nil
}

