package main

import (
	"crypto/md5"
	"crypto/sha256"
	"time"

	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Block struct {
	Pos       int
	Data      BookCheckout
	Timestamp string
	Hash      string
	PrevHash  string
}

type BookCheckout struct {
	BookID       string `json:"book_id"`
	UserID       string `json:"user_id"`
	CheckoutDate string `json:"checkout_date"`
	IsGenesis    bool   `json:"is_genesis"`
}

type Book struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	PublishDate string `json:"publish_date"`
	ISBN        string `json:"isbn"`
}

type Blockchain struct {
	blocks []*Block
}

var BlockChain *Blockchain

func validBlock(block, prevBlock *Block) bool {

	if prevBlock.Hash != block.PrevHash {
		return false
	}

	if !block.validHash(block.Hash) {
		return false
	}

	if prevBlock.Pos+1 != block.Pos {
		return false
	}
	return true
}

func (b *Block) validHash(hash string) bool {
	b.generateHash()
	return b.Hash == hash
}

func (bc *Blockchain) AddBlock(data BookCheckout) {
	prevBlock := bc.blocks[len(bc.blocks)-1]
	newBlock := CreateBlock(data, prevBlock)
	if validBlock(newBlock, prevBlock) {
		bc.blocks = append(bc.blocks, newBlock)
	}

}
func (b *Block) generateHash() {
	bytes, _ := json.Marshal(b.Data)
	data := string(b.Pos) + b.Timestamp + string(bytes) + b.PrevHash

	// Learn this part of creating a buffer and using it to store hash data
	hash := sha256.New()
	hash.Write([]byte(data))
	b.Hash = hex.EncodeToString(hash.Sum(nil))

}

func CreateBlock(data BookCheckout, prevBlock *Block) *Block {
	block := &Block{}
	block.Pos = prevBlock.Pos + 1
	block.PrevHash = prevBlock.Hash
	block.Timestamp = time.Now().String()
	block.Data = data
	block.generateHash()
	return block
}

// w -> response
// r -> request

func writeBlock(w http.ResponseWriter, r *http.Request) {
	var checkoutitem BookCheckout

	if err := json.NewDecoder(r.Body).Decode(&checkoutitem); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could not create new Block: %v", err)
		w.Write([]byte("Could not create new Block"))
		return
	}
	BlockChain.AddBlock(checkoutitem)
	resp, err := json.MarshalIndent(checkoutitem, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte("could not write block"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func newBook(w http.ResponseWriter, r *http.Request) {
	var book Book
	// The process of converting from and to struct to json, can be called encoding and decoding or Marshal and Unmarshal
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could not create new Book: %v", err)
		w.Write([]byte("Could not create new Book"))
		return
	}

	h := md5.New()
	io.WriteString(h, book.ISBN+book.PublishDate)
	book.ID = fmt.Sprintf("%x", h.Sum(nil))
	resp, err := json.MarshalIndent(book, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could not create new Book: %v", err)
		w.Write([]byte("Could not create new Book"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)

}

func GenesisBlock() *Block {
	return CreateBlock(BookCheckout{IsGenesis: true}, &Block{})
}

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{GenesisBlock()}}
}


// I didn't understand this part
func getBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(BlockChain.blocks, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could not get Blockchain: %v", err)
		w.Write([]byte("Could not get Blockchain"))
		return
	}
	io.WriteString(w, string(bytes))
}

func main() {
	BlockChain = NewBlockchain()

	r := mux.NewRouter()
	r.HandleFunc("/blockchain", getBlockchain).Methods("GET")
	r.HandleFunc("/new_checkout", writeBlock).Methods("POST")
	r.HandleFunc("/new_book", newBook).Methods("POST")

	go func() {

		for _, block := range BlockChain.blocks {
			fmt.Printf("Prev. hash: %x\n", block.PrevHash)
			bytes, _ := json.MarshalIndent(block.Data, "", " ")
			fmt.Printf("Data: %v\n", string(bytes))
			fmt.Printf("Hash: %x\n", block.Hash)
			fmt.Println()
		}

	}()
	log.Println("Listening on port 3000")
	log.Fatal(http.ListenAndServe(":3000", r))

}
