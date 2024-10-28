package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

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
	mu     sync.Mutex // For thread-safe access
}

var BlockChain *Blockchain
var Books []Book // Global variable to store all books

// Block validation
func validBlock(block, prevBlock *Block) bool {
	return prevBlock.Hash == block.PrevHash && prevBlock.Pos+1 == block.Pos && block.Hash == block.generateHash()
}

// Generates a hash for the block
func (b *Block) generateHash() string {
	data, _ := json.Marshal(b.Data)
	input := fmt.Sprintf("%d%s%s%s", b.Pos, b.Timestamp, string(data), b.PrevHash)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func (bc *Blockchain) AddBlock(data BookCheckout) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	prevBlock := bc.blocks[len(bc.blocks)-1]
	newBlock := CreateBlock(data, prevBlock)
	if validBlock(newBlock, prevBlock) {
		bc.blocks = append(bc.blocks, newBlock)
	}
}

func CreateBlock(data BookCheckout, prevBlock *Block) *Block {
	block := &Block{
		Pos:       prevBlock.Pos + 1,
		PrevHash:  prevBlock.Hash,
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      data,
	}
	block.Hash = block.generateHash()
	return block
}

func GenesisBlock() *Block {
	genesisData := BookCheckout{IsGenesis: true}
	genesisBlock := &Block{
		Pos:       0,
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      genesisData,
	}
	genesisBlock.Hash = genesisBlock.generateHash()
	return genesisBlock
}

func NewBlockchain() *Blockchain {
	return &Blockchain{blocks: []*Block{GenesisBlock()}}
}

// Handler for adding a new checkout block
func writeBlock(w http.ResponseWriter, r *http.Request) {
	var checkout BookCheckout
	if err := json.NewDecoder(r.Body).Decode(&checkout); err != nil {
		http.Error(w, "Could not create new Block", http.StatusInternalServerError)
		return
	}
	BlockChain.AddBlock(checkout)
	resp, _ := json.MarshalIndent(checkout, "", " ")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// Handler for adding a new book
func newBook(w http.ResponseWriter, r *http.Request) {
	var book Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		http.Error(w, "Could not create new Book", http.StatusInternalServerError)
		return
	}

	// Create a unique ID for the book using SHA-256
	hash := sha256.New()
	io.WriteString(hash, book.ISBN+book.PublishDate)
	book.ID = hex.EncodeToString(hash.Sum(nil))

	// Store the new book in the global Books slice
	Books = append(Books, book)

	resp, err := json.MarshalIndent(book, "", "  ")
	if err != nil {
		http.Error(w, "Could not create new Book", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// Handler for viewing all books
func getAllBooks(w http.ResponseWriter, r *http.Request) {
	resp, err := json.MarshalIndent(Books, "", "  ")
	if err != nil {
		http.Error(w, "Could not retrieve books", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

// Handler for viewing the blockchain
func getBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(BlockChain.blocks, "", "  ")
	if err != nil {
		http.Error(w, "Could not get Blockchain", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func main() {
	BlockChain = NewBlockchain()
	r := mux.NewRouter()
	r.HandleFunc("/blockchain", getBlockchain).Methods("GET")
	r.HandleFunc("/new_checkout", writeBlock).Methods("POST")
	r.HandleFunc("/new_book", newBook).Methods("POST")
	r.HandleFunc("/all_books", getAllBooks).Methods("GET") // New route for retrieving all books

	log.Println("Listening on port 3000")
	log.Fatal(http.ListenAndServe(":3000", r))
}
