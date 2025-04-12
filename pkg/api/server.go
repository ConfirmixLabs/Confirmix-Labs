package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"time"
	"encoding/hex"

	"github.com/gorilla/mux"
	"github.com/google/uuid"
	"confirmix/pkg/blockchain"
	"confirmix/pkg/consensus"
	"confirmix/pkg/types"
)

// WebServer represents the web server instance
type WebServer struct {
	blockchain      *blockchain.Blockchain
	consensusEngine *consensus.HybridConsensus
	validatorManager *consensus.ValidatorManager
	governance      *consensus.Governance
	port           int
	router         *mux.Router
	server         *http.Server  // Add server field
	
	// Önbellek verileri
	validatorsCache      []blockchain.ValidatorInfo
	validatorsCacheTime  time.Time
	validatorsCacheMutex sync.RWMutex
	
	// İşlemler için önbellek
	transactionsCache      []*blockchain.Transaction
	transactionsCacheTime  time.Time
	transactionsCacheMutex sync.RWMutex
	
	// Bekleyen işlemler için ayrı önbellek
	pendingTxCache      []*blockchain.Transaction
	pendingTxCacheTime  time.Time
	pendingTxCacheMutex sync.RWMutex
	
	// Onaylanmış işlemler için ayrı önbellek
	confirmedTxCache      []*blockchain.Transaction
	confirmedTxCacheTime  time.Time
	confirmedTxCacheMutex sync.RWMutex
	
	// Genel blockchain önbellekleri
	
	// Blok önbelleği - key: block index, value: *blockchain.Block
	blockCache       sync.Map // thread-safe map
	blockCacheExpiry sync.Map // ne zaman süresi dolacak
	
	// Bakiye önbelleği - key: address, value: *big.Int
	balanceCache       sync.Map
	balanceCacheExpiry sync.Map
}

// NewWebServer creates a new web server instance
func NewWebServer(bc *blockchain.Blockchain, ce *consensus.HybridConsensus, vm *consensus.ValidatorManager, gov *consensus.Governance, port int) *WebServer {
	ws := &WebServer{
		blockchain:      bc,
		consensusEngine: ce,
		validatorManager: vm,
		governance:      gov,
		port:           port,
		router:         mux.NewRouter(),
	}
	ws.setupRoutes()
	return ws
}

// enableCORS enables CORS for all routes
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// setupRoutes configures the HTTP routes
func (ws *WebServer) setupRoutes() {
	ws.router = mux.NewRouter()
	
	// Enable CORS for all routes
	ws.router.Use(enableCORS)

	// Blockchain routes
	ws.router.HandleFunc("/api/status", ws.getStatus).Methods("GET")
	ws.router.HandleFunc("/api/blocks", ws.getBlocks).Methods("GET")
	ws.router.HandleFunc("/api/blocks/{index}", ws.getBlockByIndex).Methods("GET")
	ws.router.HandleFunc("/api/transactions", ws.getAllTransactions).Methods("GET")
	ws.router.HandleFunc("/api/transactions/pending", ws.getPendingTransactions).Methods("GET")
	ws.router.HandleFunc("/api/transactions/confirmed", ws.getConfirmedTransactions).Methods("GET")
	ws.router.HandleFunc("/api/transactions", ws.createTransaction).Methods("POST")
	ws.router.HandleFunc("/api/blockchain/transactions/{hash}/revert", ws.revertTransaction).Methods("POST")
	
	// Wallet routes
	ws.router.HandleFunc("/api/wallet/create", ws.createWallet).Methods("POST")
	ws.router.HandleFunc("/api/wallet/import", ws.importWallet).Methods("POST")
	ws.router.HandleFunc("/api/wallet/balance/{address}", ws.getWalletBalance).Methods("GET")
	ws.router.HandleFunc("/api/wallet/balance/{address}/simple", ws.getWalletBalanceSimple).Methods("GET")
	ws.router.HandleFunc("/api/wallet/transfer", ws.transfer).Methods("POST")
	
	// Mining routes
	ws.router.HandleFunc("/api/mine", ws.mineBlock).Methods("POST")
	
	// Validator routes
	ws.router.HandleFunc("/api/validators", ws.getValidators).Methods("GET")
	ws.router.HandleFunc("/api/validators/register", ws.registerValidator).Methods("POST")
	ws.router.HandleFunc("/api/validators/approve", ws.approveValidator).Methods("POST")
	ws.router.HandleFunc("/api/validators/reject", ws.rejectValidator).Methods("POST")
	ws.router.HandleFunc("/api/validators/suspend", ws.suspendValidator).Methods("POST")
	
	// Admin routes
	ws.router.HandleFunc("/api/admin/add", ws.addAdmin).Methods("POST")
	ws.router.HandleFunc("/api/admin/remove", ws.removeAdmin).Methods("POST")
	ws.router.HandleFunc("/api/admin/list", ws.listAdmins).Methods("GET")
	
	// Governance routes
	ws.router.HandleFunc("/api/proposals", ws.listProposals).Methods("GET")
	ws.router.HandleFunc("/api/proposals/{id}", ws.getProposal).Methods("GET")
	ws.router.HandleFunc("/api/proposals/create", ws.createProposal).Methods("POST")
	ws.router.HandleFunc("/api/proposals/vote", ws.castVote).Methods("POST")
	
	// Health check
	ws.router.HandleFunc("/api/health", ws.getHealthCheck).Methods("GET")

	// Multi-signature routes
	ws.router.HandleFunc("/api/multisig/wallet/create", ws.createMultisigWallet).Methods("POST", "OPTIONS")
	ws.router.HandleFunc("/api/multisig/wallet/{address}", ws.getMultiSigWallet).Methods("GET", "OPTIONS")
	ws.router.HandleFunc("/api/multisig/transaction/create", ws.createMultiSigTransaction).Methods("POST", "OPTIONS")
	ws.router.HandleFunc("/api/multisig/transaction/sign", ws.signMultisigTransaction).Methods("POST", "OPTIONS")
	ws.router.HandleFunc("/api/multisig/transaction/execute", ws.executeMultisigTransaction).Methods("POST", "OPTIONS")
	ws.router.HandleFunc("/api/multisig/transaction/{walletAddress}/{txID}/status", ws.getMultiSigTransactionStatus).Methods("GET", "OPTIONS")
	ws.router.HandleFunc("/api/multisig/transaction/{walletAddress}/pending", ws.getMultiSigPendingTransactions).Methods("GET", "OPTIONS")
}

// Start starts the web server
func (ws *WebServer) Start() error {
	// Preload cache data
	log.Printf("Preloading caches for better performance...")
	ws.PreloadCache()
	
	// Start the server
	addr := fmt.Sprintf(":%d", ws.port)
	log.Printf("Web server listening on %s", addr)
	
	// Create and store the server instance
	ws.server = &http.Server{
		Addr:    addr,
		Handler: ws.router,
	}
	
	return ws.server.ListenAndServe()
}

// PreloadCache pre-populates cache to avoid initial timeouts
func (ws *WebServer) PreloadCache() {
	log.Printf("Starting cache preloading (simplified)...")
	startTime := time.Now()
	
	// Önce blokzincirdeki önemli adresleri alarak bakiye önbelleğini dolduralım
	addresses := ws.blockchain.GetAllAddresses()
	if len(addresses) > 0 {
		log.Printf("Found %d addresses in blockchain (including genesis and node address)", len(addresses))
		
		// Limit the number of addresses to preload
		preloadCount := 10
		if len(addresses) < preloadCount {
			preloadCount = len(addresses)
		}
		
		preloadAddresses := addresses[:preloadCount]
		
		// Create static cache data
		for _, addr := range preloadAddresses {
			// Default balance 0 tokens
			ws.balanceCache.Store(addr, big.NewInt(0))
			ws.balanceCacheExpiry.Store(addr, time.Now().Add(60*time.Second))
		}
		
		log.Printf("Preloaded %d address balances with default value in %v", 
			preloadCount, time.Since(startTime))
	} else {
		log.Printf("No addresses found to preload balances for")
	}
	
	// Validators - Set a static default list first for immediate use
	defaultValidators := []blockchain.ValidatorInfo{}
	
	// First fill the cache with default values
	ws.validatorsCacheMutex.Lock()
	ws.validatorsCache = defaultValidators 
	ws.validatorsCacheTime = time.Now()
	ws.validatorsCacheMutex.Unlock()
	
	// Try to get real validators in the background
	go func() {
		validatorStart := time.Now()
		log.Printf("Background fetching registered validators...")
		
		// Try with a short timeout - but in background to not delay page load
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		
		// Use a different channel for communication
		done := make(chan bool, 1)
		var validators []blockchain.ValidatorInfo
		
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC in validator preloading: %v", r)
				}
				done <- true
			}()
			
			// Get validators from blockchain
			validators = ws.blockchain.GetValidators()
			
			if len(validators) > 0 {
				// Update cache
				ws.validatorsCacheMutex.Lock()
				ws.validatorsCache = validators
				ws.validatorsCacheTime = time.Now()
				ws.validatorsCacheMutex.Unlock()
				
				log.Printf("Background loaded %d registered validators in %v", 
					len(validators), time.Since(validatorStart))
			} else {
				log.Printf("No registered validators found in blockchain")
			}
		}()
		
		// Wait for either completion or timeout
		select {
		case <-done:
			// İşlem tamamlandı
		case <-ctx.Done():
			log.Printf("Timeout preloading validators: %v", ctx.Err())
		}
	}()
	
	// Boş transaction listeleri oluşturalım - sonra arkaplanda gerçekleri almayı deneriz
	emptyTxs := make([]*blockchain.Transaction, 0)
	
	// Boş bekleyen işlem listesi
	ws.pendingTxCacheMutex.Lock()
	ws.pendingTxCache = emptyTxs
	ws.pendingTxCacheTime = time.Now()
	ws.pendingTxCacheMutex.Unlock()
	
	// Boş onaylanmış işlem listesi
	ws.confirmedTxCacheMutex.Lock()
	ws.confirmedTxCache = emptyTxs  
	ws.confirmedTxCacheTime = time.Now()
	ws.confirmedTxCacheMutex.Unlock()
	
	// Boş tüm işlemler listesi
	ws.transactionsCacheMutex.Lock()
	ws.transactionsCache = emptyTxs
	ws.transactionsCacheTime = time.Now()
	ws.transactionsCacheMutex.Unlock()
	
	log.Printf("Initialized empty transaction lists")
	
	// Arkaplanda işlemleri getirmeye çalışalım
	go func() {
		txStart := time.Now()
		
		// Bekleyen işlemleri al
		pending := ws.blockchain.GetPendingTransactions()
		pendingWithStatus := make([]*blockchain.Transaction, 0, len(pending))
		
		for _, tx := range pending {
			txCopy := *tx
			txCopy.Status = "pending"
			pendingWithStatus = append(pendingWithStatus, &txCopy)
		}
		
		// Sadece boş değilse güncelle
		if len(pendingWithStatus) > 0 {
			ws.pendingTxCacheMutex.Lock()
			ws.pendingTxCache = pendingWithStatus
			ws.pendingTxCacheTime = time.Now()
			ws.pendingTxCacheMutex.Unlock()
			
			log.Printf("Background loaded %d pending transactions in %v", 
				len(pendingWithStatus), time.Since(txStart))
				
			// Combined transactions listesini de güncelle
			ws.transactionsCacheMutex.Lock()
			ws.transactionsCache = pendingWithStatus // Başlangıç için sadece bekleyenler
			ws.transactionsCacheTime = time.Now()
			ws.transactionsCacheMutex.Unlock()
		}
		
		// İşimiz bitti, onaylanmış işlemleri daha sonra lazım olursa getireceğiz
		log.Printf("Transaction preloading completed in %v", time.Since(txStart))
	}()
	
	log.Printf("Cache preloading initiated in %v", time.Since(startTime))
}

// getStatus handles the status endpoint
func (ws *WebServer) getStatus(w http.ResponseWriter, r *http.Request) {
	// Set headers for CORS
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Create a static status response
	// Using cached or default values to avoid blockchain calls
	status := struct {
		Status   string `json:"status"`
		Height   uint64 `json:"height"`
		Uptime   string `json:"uptime"`
		Version  string `json:"version"`
		NodeType string `json:"nodeType"`
	}{
		Status:   "online",
		Height:   ws.blockchain.GetChainHeight(),
		Uptime:   "active",
		Version:  "1.0.0",
		NodeType: "validator",
	}
	
	// Always return OK
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// getBlocks handles the blocks endpoint
func (ws *WebServer) getBlocks(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse limit from query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // Default limit
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	
	// Cap limit at 50
	if limit > 50 {
		limit = 50
	}
	
	// Set a timeout for the handler
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	blocksChan := make(chan []struct {
		Index        uint64 `json:"Index"`
		Timestamp    int64  `json:"Timestamp"`
		Hash         string `json:"Hash"`
		PrevHash     string `json:"PrevHash"`
		Validator    string `json:"Validator"`
		Transactions int    `json:"Transactions"`
	}, 1)
	
	// Do the work in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in getBlocks: %v", r)
			}
			done <- true
		}()
		
		log.Printf("Getting blocks from blockchain, limit=%d", limit)
		
		// Get chain height safely as int (not uint64)
		chainHeight := int(ws.blockchain.GetChainHeight())
		
		// Create result array
		blocksResponse := make([]struct {
			Index        uint64 `json:"Index"`
			Timestamp    int64  `json:"Timestamp"`
			Hash         string `json:"Hash"`
			PrevHash     string `json:"PrevHash"`
			Validator    string `json:"Validator"`
			Transactions int    `json:"Transactions"`
		}, 0, limit)
		
		// Start from the most recent block and go backwards
		// Ensure we don't go negative or exceed the chain height
		for i := chainHeight; i >= 0 && len(blocksResponse) < limit; i-- {
			// Convert index to uint64 only when passing to blockchain API
			blockIndex := uint64(i)
			blockIndexKey := fmt.Sprintf("block_%d", blockIndex)
			
			var block *blockchain.Block
			var err error
			
			// First check cache
			if cachedValue, ok := ws.blockCache.Load(blockIndexKey); ok {
				if expiryTime, ok := ws.blockCacheExpiry.Load(blockIndexKey); ok {
					if time.Now().Before(expiryTime.(time.Time)) {
						// Cached value is still valid
						block = cachedValue.(*blockchain.Block)
					}
				}
			}
			
			// If not in cache, get from blockchain
			if block == nil {
				block, err = ws.blockchain.GetBlockByIndex(blockIndex)
				if err != nil {
					log.Printf("Error getting block at index %d: %v", i, err)
					continue
				}
				
				// Cache block for future use (blocks don't change)
				ws.blockCache.Store(blockIndexKey, block)
				ws.blockCacheExpiry.Store(blockIndexKey, time.Now().Add(60*time.Second))
			}
			
			// Make sure block has valid Hash field
			if block != nil {
				blockHash := block.Hash
				if blockHash == "" {
					// Generate a hash if missing
					blockHash = fmt.Sprintf("block_%d_%d", block.Index, block.Timestamp)
				}
				
				blockResp := struct {
					Index        uint64 `json:"Index"`
					Timestamp    int64  `json:"Timestamp"`
					Hash         string `json:"Hash"`
					PrevHash     string `json:"PrevHash"`
					Validator    string `json:"Validator"`
					Transactions int    `json:"Transactions"`
				}{
					Index:        block.Index,
					Timestamp:    block.Timestamp,
					Hash:         blockHash,
					PrevHash:     block.PrevHash,
					Validator:    block.Validator,
					Transactions: len(block.Transactions),
				}
				
				blocksResponse = append(blocksResponse, blockResp)
			}
		}
		
		log.Printf("Retrieved %d blocks", len(blocksResponse))
		blocksChan <- blocksResponse
	}()
	
	// Wait for completion or timeout
	select {
	case blocks := <-blocksChan:
		w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(blocks)
		
	case <-done:
		// No blocks sent, return empty array
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
		
	case <-ctx.Done():
		// Timeout - return what we have
		log.Printf("Timeout getting blocks: %v", ctx.Err())
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}
}

// getPendingTransactions handles the pending transactions endpoint with caching
func (ws *WebServer) getPendingTransactions(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers and Content-Type
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Get limit parameter, default to 50
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			// Cap the limit to prevent performance issues
			if l > 100 {
				l = 100
			}
			limit = l
		}
	}
	
	// Önbellekteki verileri kontrol edelim (5 saniyeden daha yeni ise)
	ws.pendingTxCacheMutex.RLock()
	cacheAge := time.Since(ws.pendingTxCacheTime)
	hasCache := len(ws.pendingTxCache) > 0 && cacheAge < 5*time.Second
	
	// Eğer önbellekte güncel veri varsa ve istenen limit önbellek boyutundan az veya eşitse, hemen döndürelim
	if hasCache && limit <= len(ws.pendingTxCache) {
		// Önbellekten limiti kadar veri alalım
		txs := ws.pendingTxCache
		if limit < len(txs) {
			txs = txs[:limit]
		}
		ws.pendingTxCacheMutex.RUnlock()
		
		log.Printf("Returning %d pending transactions from cache (age: %v)", len(txs), cacheAge)
		w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(txs)
		return
	}
	ws.pendingTxCacheMutex.RUnlock()
	
	// Önbellekte veri yoksa veya eski ise veya istenen limit önbellek boyutundan büyükse, yeni veri alalım
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second) // Timeout süresini 3 saniyeden 10 saniyeye çıkardık
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	var pendingTxs []*blockchain.Transaction
	var err error
	
	// Do the work in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in getPendingTransactions: %v", r)
				err = fmt.Errorf("internal error: %v", r)
			}
			done <- true
		}()
		
		startTime := time.Now()
		log.Printf("Getting pending transactions from blockchain")
		
		// Get all pending transactions
		pendingTxsRaw := ws.blockchain.GetPendingTransactions()
		
		// Create a new array to hold our response and make a deep copy to avoid race conditions
		pendingTxs = make([]*blockchain.Transaction, 0, len(pendingTxsRaw))
		
		for _, tx := range pendingTxsRaw {
			// Create a copy of each transaction
			txCopy := *tx
			// Add a status field for pending transactions
			txCopy.Status = "pending"
			pendingTxs = append(pendingTxs, &txCopy)
		}
		
		// Önbelleği güncelle
		ws.pendingTxCacheMutex.Lock()
		ws.pendingTxCache = pendingTxs
		ws.pendingTxCacheTime = time.Now()
		ws.pendingTxCacheMutex.Unlock()
		
		log.Printf("Retrieved %d pending transactions in %v", len(pendingTxs), time.Since(startTime))
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		if err != nil {
			log.Printf("Error getting pending transactions: %v", err)
			// Return an empty array instead of an error
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(make([]*blockchain.Transaction, 0))
			return
		}
		
		// Create empty array if nil
		if pendingTxs == nil {
			pendingTxs = make([]*blockchain.Transaction, 0)
		}
		
		// Limit the response if needed
		if len(pendingTxs) > limit {
			pendingTxs = pendingTxs[:limit]
		}
		
		// Return the transactions as JSON
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(pendingTxs)
		if err != nil {
			log.Printf("Error encoding pending transactions: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
		return
		
	case <-ctx.Done():
		log.Printf("Timeout getting pending transactions: %v", ctx.Err())
		
		// Önbellekte herhangi bir veri varsa, eski de olsa döndürelim
		ws.pendingTxCacheMutex.RLock()
		hasCacheData := len(ws.pendingTxCache) > 0
		cachedTxs := ws.pendingTxCache // Kopyasını alalım
		if limit < len(cachedTxs) {
			cachedTxs = cachedTxs[:limit]
		}
		ws.pendingTxCacheMutex.RUnlock()
		
		if hasCacheData {
			log.Printf("Returning %d pending transactions from stale cache due to timeout", len(cachedTxs))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(cachedTxs)
			return
		}
		
		// Hiç önbellek verisi yoksa boş dizi döndür
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(make([]*blockchain.Transaction, 0))
	}
}

// createTransaction handles the transaction creation endpoint
func (ws *WebServer) createTransaction(w http.ResponseWriter, r *http.Request) {
	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Set a timeout for the handler
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	var simpleTransaction *blockchain.Transaction
	var err error
	
	// Do the work in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in createTransaction: %v", r)
				err = fmt.Errorf("internal error: %v", r)
			}
			done <- true
		}()
		
		// Log the request body for debugging
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			err = fmt.Errorf("error reading request: %v", err)
			return
		}
		r.Body.Close()
		
		// Create a new reader with the same bytes for json.Decode
		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		log.Printf("Received transaction request: %s", string(bodyBytes))
		
		var tx struct {
			From  string `json:"from"`
			To    string `json:"to"`
			Value uint64 `json:"value"`
			Data  string `json:"data,omitempty"`
		}
		
		if err = json.NewDecoder(r.Body).Decode(&tx); err != nil {
			log.Printf("Transaction decode error: %v", err)
			err = fmt.Errorf("invalid transaction format: %v", err)
			return
		}
		
		// Temel doğrulama kontrolleri
		if tx.From == "" {
			err = errors.New("sender address cannot be empty")
			return
		}
		
		if tx.To == "" {
			err = errors.New("recipient address cannot be empty")
			return
		}
		
		if tx.Value <= 0 {
			err = fmt.Errorf("invalid transaction amount: %d", tx.Value)
			return
		}
		
		// Debug logging
		log.Printf("Creating transaction: From=%s, To=%s, Value=%d", tx.From, tx.To, tx.Value)
		
		// Ayrıca kullanıcının bekleyen diğer işlemlerini de kontrol edelim
		pendingTxs := ws.blockchain.GetPendingTransactions()
		pendingSpend := uint64(0)
		
		for _, pendingTx := range pendingTxs {
			if pendingTx.From == tx.From {
				pendingSpend += pendingTx.Value
			}
		}
		
		// Get sender balance
		senderBalanceBigInt, err := ws.blockchain.GetBalance(tx.From)
		if err != nil {
			log.Printf("Error getting balance for sender %s: %v", tx.From, err)
			err = fmt.Errorf("cannot get sender balance: %v", err)
			return
		}

		// Check if sender balance can be represented as uint64
		if !senderBalanceBigInt.IsUint64() {
			// Balance is too large for uint64, assume sufficient for this check
			// Continue processing
		} else {
			senderBalance := senderBalanceBigInt.Uint64()
			
			// Toplam harcama = bekleyen harcamalar + yeni işlem
			totalSpend := pendingSpend + tx.Value
			
			if totalSpend > senderBalance {
				log.Printf("Insufficient balance for transaction: required=%d, available=%d, pending=%d, total=%d", 
					tx.Value, senderBalance, pendingSpend, totalSpend)
				err = fmt.Errorf("insufficient balance: required=%d, available=%d, pending=%d", 
					tx.Value, senderBalance, pendingSpend)
				return
			}
		}
		
		// Create a simple transaction
		simpleTransaction = &blockchain.Transaction{
			ID:        fmt.Sprintf("%x", time.Now().UnixNano()),
			From:      tx.From,
			To:        tx.To,
			Value:     tx.Value,
			Timestamp: time.Now().Unix(),
			Type:      "regular",
		}
		
		// Data handling
		if tx.Data != "" {
			simpleTransaction.Data = []byte(tx.Data)
		}
		
		// Add transaction to pool
		if err = ws.blockchain.AddTransaction(simpleTransaction); err != nil {
			log.Printf("Error adding transaction to pool: %v", err)
			return
		}
		
		log.Printf("Transaction added to pool: %s", simpleTransaction.ID)
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		if err != nil {
			log.Printf("Transaction error: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		// Return the transaction
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"transaction": simpleTransaction,
		})
		
	case <-ctx.Done():
		log.Printf("Timeout creating transaction: %v", ctx.Err())
		w.WriteHeader(http.StatusGatewayTimeout)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Request timed out",
		})
	}
}

// createWallet handles the wallet creation endpoint
func (ws *WebServer) createWallet(w http.ResponseWriter, r *http.Request) {
	// Automatically handle CORS preflight request
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	var response struct {
		Address    string `json:"address"`
		PublicKey  string `json:"publicKey"`
		PrivateKey string `json:"privateKey"`
		
		Balance    uint64 `json:"balance"`
		Success    bool   `json:"success"`
	}
	var err error
	
	// Do the wallet creation in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in createWallet: %v", r)
				err = fmt.Errorf("internal error: %v", r)
			}
			done <- true
		}()
		
		start := time.Now()
		log.Printf("Starting wallet creation")
		
		// Create wallet
	wallet, err := blockchain.CreateWallet()
	if err != nil {
			log.Printf("Failed to create wallet: %v", err)
			err = fmt.Errorf("failed to create wallet: %v", err)
		return
	}
		
		log.Printf("Wallet created with address: %s", wallet.Address)
		
		// Prepare initial response
		response = struct {
			Address    string `json:"address"`
			PublicKey  string `json:"publicKey"`
			PrivateKey string `json:"privateKey"`
			
			Balance    uint64 `json:"balance"`
			Success    bool   `json:"success"`
		}{
			Address:    wallet.Address,
			PublicKey:  wallet.KeyPair.GetPublicKeyString(),
			PrivateKey: wallet.KeyPair.GetPrivateKeyString(),
			Balance:    0, // Start with 0 balance
			Success:    true,
	}
	
	// Save wallet's key pair to blockchain
	ws.blockchain.AddKeyPair(wallet.Address, wallet.KeyPair)
		log.Printf("Key pair added for address: %s", wallet.Address)

		// Create account with 0 initial balance
		initialBalance := big.NewInt(0)
		if err := ws.blockchain.CreateAccount(wallet.Address, initialBalance); err != nil {
			log.Printf("Warning: Error creating account: %v", err)
		} else {
			log.Printf("Account created with initial balance: 0 tokens")
			
			// Pre-cache the balance
			ws.balanceCache.Store(wallet.Address, initialBalance)
			ws.balanceCacheExpiry.Store(wallet.Address, time.Now().Add(60*time.Second))
		}
		
		// Save blockchain state to disk after creating a wallet
		ws.blockchain.SaveToDisk()
		log.Printf("Blockchain state saved to disk after wallet creation")
		
		log.Printf("Wallet created successfully in %v", time.Since(start))
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		if err != nil {
			log.Printf("Error in wallet creation: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// Return the wallet information
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
		
	case <-ctx.Done():
		log.Printf("Timeout creating wallet: %v", ctx.Err())
		http.Error(w, "Wallet creation timed out", http.StatusGatewayTimeout)
	}
}

// getWalletBalance handles the wallet balance endpoint
func (ws *WebServer) getWalletBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	// Get raw balance
	balance, err := ws.blockchain.GetBalance(address)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get balance: %v", err), http.StatusInternalServerError)
		return
	}

	// Return balance as string
	response := map[string]interface{}{
		"address": address,
		"balance": balance.String(),
	}

	json.NewEncoder(w).Encode(response)
}

// mineBlock handles the mining endpoint
func (ws *WebServer) mineBlock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		Validator string `json:"validator"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding mining request: %v", err)
		http.Error(w, fmt.Sprintf("invalid request format: %v", err), http.StatusBadRequest)
		return
	}
	
	if req.Validator == "" {
		log.Printf("Mining request error: validator address is empty")
		http.Error(w, "validator address is required", http.StatusBadRequest)
		return
	}
	
	// Log the mining attempt
	log.Printf("Mining attempt from address: %s", req.Validator)
	
	// Check if the address is a registered validator
	if !ws.blockchain.IsValidator(req.Validator) {
		log.Printf("Unauthorized mining attempt from non-validator address: %s", req.Validator)
		http.Error(w, fmt.Sprintf("address %s is not a registered validator", req.Validator), http.StatusUnauthorized)
		return
	}
	
	// Get validator's key pair
	validators := ws.blockchain.GetValidators()
	if len(validators) == 0 {
		log.Printf("No validators found in the blockchain")
		http.Error(w, "no validators available", http.StatusBadRequest)
		return
	}

	validatorAddress := validators[0].Address
	keyPair, exists := ws.blockchain.GetKeyPair(validatorAddress)
	if !exists {
		log.Printf("Key pair not found for validator: %s", validatorAddress)
		
		// List all available addresses for debugging
		addresses := ws.blockchain.GetAllAddresses()
		log.Printf("Available addresses in blockchain: %v", addresses)
		
		http.Error(w, fmt.Sprintf("validator's key pair not found for %s", validatorAddress), http.StatusBadRequest)
		return
	}
	log.Printf("Retrieved key pair for validator: %s", validatorAddress)
	
	// Get validator's human proof
	humanProof := ws.blockchain.GetHumanProof(req.Validator)
	if humanProof == "" {
		log.Printf("Human proof not found for validator: %s", req.Validator)
		http.Error(w, "validator's human proof not found", http.StatusBadRequest)
		return
	}
	log.Printf("Retrieved human proof for validator: %s", req.Validator)
	
	// Get pending transactions
	pendingTxs := ws.blockchain.GetPendingTransactions()
	log.Printf("Retrieved %d pending transactions", len(pendingTxs))
	
	if len(pendingTxs) == 0 {
		log.Printf("No pending transactions to mine for validator: %s", req.Validator)
		http.Error(w, "no pending transactions to mine", http.StatusBadRequest)
		return
	}
	
	// Validate and pre-process all transactions
	validTxs := []*blockchain.Transaction{}
	invalidTxs := []*blockchain.Transaction{}
	
	// Her bir göndericinin bloktaki tüm işlemler sonrası toplam harcamasını takip edelim
	senderSpends := make(map[string]uint64)
	senderBalances := make(map[string]uint64)
	
	for _, tx := range pendingTxs {
		// Validate transaction basics
		if tx.From == "" || tx.To == "" || tx.Value <= 0 {
			log.Printf("Invalid transaction found: From=%s, To=%s, Value=%d", tx.From, tx.To, tx.Value)
			invalidTxs = append(invalidTxs, tx)
			continue
		}
		
		// Get sender balance if not already cached
		if _, exists := senderBalances[tx.From]; !exists {
			balance, err := ws.blockchain.GetBalance(tx.From)
		if err != nil {
			log.Printf("Warning: Cannot get balance for sender %s: %v", tx.From, err)
			invalidTxs = append(invalidTxs, tx)
			continue
			}
			senderBalances[tx.From] = balance.Uint64()
		}
		
		// Calculate total spend for this sender so far
		if _, exists := senderSpends[tx.From]; !exists {
			senderSpends[tx.From] = 0
		}
		
		totalSpentBySender := senderSpends[tx.From] + tx.Value
		
		// Check if sender has enough balance considering all transactions in this block
		senderBalance := senderBalances[tx.From]
		
		if totalSpentBySender > senderBalance {
			log.Printf("Warning: Insufficient balance for transaction %s after considering previous txs in block (sender: %s, amount: %d, balance: %d, total spent: %d)",
				tx.ID, tx.From, tx.Value, senderBalance, totalSpentBySender)
			invalidTxs = append(invalidTxs, tx)
			continue
		}
		
		// Verify transaction signature
		if !tx.SimpleVerify() {
			log.Printf("Warning: Transaction %s has invalid signature", tx.ID)
			invalidTxs = append(invalidTxs, tx)
			continue
		}
		
		// Bu işlem geçerli, toplam harcamayı güncelleyelim
		senderSpends[tx.From] = totalSpentBySender
		validTxs = append(validTxs, tx)
		log.Printf("Valid transaction found: ID=%s, From=%s, To=%s, Value=%d", 
			tx.ID, tx.From, tx.To, tx.Value)
	}
	
	if len(validTxs) == 0 {
		log.Printf("No valid transactions to mine for validator: %s", req.Validator)
		http.Error(w, "no valid transactions to mine", http.StatusBadRequest)
		return
	}
	
	log.Printf("Found %d valid transactions out of %d pending transactions", len(validTxs), len(pendingTxs))
	
	// Create a new block
	lastBlock := ws.blockchain.GetLatestBlock()
	
	// Log latest block details
	log.Printf("Latest block: Index=%d, Hash=%s", lastBlock.Index, lastBlock.Hash)
	
	newBlock := &blockchain.Block{
		Index:        lastBlock.Index + 1,
		Timestamp:    time.Now().Unix(),
		Transactions: validTxs, // Only include valid transactions
		PrevHash:     lastBlock.Hash,
		Validator:    req.Validator,
		HumanProof:   humanProof, // Validatör için saklanan gerçek human proof kullanıyoruz
	}
	
	// Calculate and set the block hash
	newBlock.Hash = newBlock.CalculateHash()
	log.Printf("New block created with hash: %s", newBlock.Hash)
	
	// Sign the block
	if err := newBlock.Sign(keyPair.PrivateKey); err != nil {
		log.Printf("Error during block signing by validator %s: %v", req.Validator, err)
		http.Error(w, fmt.Sprintf("block signing failed: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Block successfully signed by validator %s", req.Validator)
	
	// Add block to blockchain
	if err := ws.blockchain.AddBlock(newBlock); err != nil {
		log.Printf("Error adding block to blockchain: %v", err)
		http.Error(w, fmt.Sprintf("failed to add block: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Block #%d successfully added to blockchain", newBlock.Index)
	
	// Remove invalid transactions from the pool
	for _, tx := range invalidTxs {
		if err := ws.blockchain.RemoveTransaction(tx.ID); err != nil {
			log.Printf("Warning: Failed to remove invalid transaction %s: %v", tx.ID, err)
		}
	}
	
	// Process valid transactions and update balances
	successfulTxs := []*blockchain.Transaction{}
	failedTxs := []*blockchain.Transaction{}
	
	for _, tx := range validTxs {
		// Final balance check before updating
		currentBalance, err := ws.blockchain.GetBalance(tx.From)
		if err != nil {
			log.Printf("Error checking balance for %s: %v", tx.From, err)
			failedTxs = append(failedTxs, tx)
			continue
		}
		
		if currentBalance.Cmp(big.NewInt(int64(tx.Value))) < 0 {
			log.Printf("Final balance check failed for tx %s: required=%d, available=%d", 
				tx.ID, tx.Value, currentBalance)
			failedTxs = append(failedTxs, tx)
			continue
		}
		
		// Update balances
		if err := ws.blockchain.UpdateBalances(tx); err != nil {
			log.Printf("Failed to update balances for transaction %s: %v", tx.ID, err)
			failedTxs = append(failedTxs, tx)
		} else {
			// Transaction successfully processed
			successfulTxs = append(successfulTxs, tx)
			log.Printf("Successfully processed transaction %s: %d tokens from %s to %s",
				tx.ID, tx.Value, tx.From, tx.To)
				
			// Get and log new balances for verification
			newSenderBalance, _ := ws.blockchain.GetBalance(tx.From)
			newReceiverBalance, _ := ws.blockchain.GetBalance(tx.To)
			log.Printf("Updated balances - Sender %s: %d, Receiver %s: %d", 
				tx.From, newSenderBalance, tx.To, newReceiverBalance)
		}
	}
	
	// Log summary
	log.Printf("Block #%d mining summary: %d successful transactions, %d failed transactions",
		newBlock.Index, len(successfulTxs), len(failedTxs))
	
	// Return block information with successful transactions
	response := struct {
		Block             *blockchain.Block          `json:"block"`
		SuccessfulTxs     []*blockchain.Transaction  `json:"successfulTransactions"`
		FailedTxs         []*blockchain.Transaction  `json:"failedTransactions"`
		InvalidTxs        int                        `json:"invalidTransactions"`
	}{
		Block:         newBlock,
		SuccessfulTxs: successfulTxs,
		FailedTxs:     failedTxs,
		InvalidTxs:    len(invalidTxs),
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// registerValidator handles the validator registration endpoint
func (ws *WebServer) registerValidator(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		Address    string `json:"address"`
		HumanProof string `json:"humanProof"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate inputs
	if req.Address == "" {
		http.Error(w, "address is required", http.StatusBadRequest)
		return
	}
	
	if req.HumanProof == "" {
		http.Error(w, "humanProof is required", http.StatusBadRequest)
		return
	}
	
	// Check if address has a key pair
	if _, exists := ws.blockchain.GetKeyPair(req.Address); !exists {
		http.Error(w, "address does not have a registered key pair", http.StatusBadRequest)
		return
	}
	
	// Check if already a validator
	if ws.blockchain.IsValidator(req.Address) {
		http.Error(w, "address is already a validator", http.StatusConflict)
		return
	}
	
	// Add as validator
	if err := ws.blockchain.AddValidator(req.Address, req.HumanProof); err != nil {
		log.Printf("Failed to register validator: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Save blockchain state to disk after registering a validator
	go ws.blockchain.SaveToDisk()
	
	log.Printf("Address %s registered as validator with proof: %s", req.Address, req.HumanProof)
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Validator registered successfully",
		"address": req.Address,
	})
}

// getValidators returns the list of registered validators with caching
func (ws *WebServer) getValidators(w http.ResponseWriter, r *http.Request) {
	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Statik validator listesi - ValidatorInfo struct'ının gerçek yapısına uygun
	// Sadece zorunlu alanlar olan Address ve HumanProof kullanılıyor
	defaultValidators := []blockchain.ValidatorInfo{}
	
	// İlk olarak önbellekteki verileri kontrol edelim (30 saniyeden daha yeni ise)
	ws.validatorsCacheMutex.RLock()
	cacheAge := time.Since(ws.validatorsCacheTime)
	hasCache := len(ws.validatorsCache) > 0 && cacheAge < 30*time.Second
	
	// Eğer önbellekte güncel veri varsa, hemen döndürelim
	if hasCache {
		validators := ws.validatorsCache // Kopyasını alalım
		ws.validatorsCacheMutex.RUnlock()
		
		log.Printf("Returning %d validators from cache (age: %v)", len(validators), cacheAge)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(validators)
		return
	}
	
	// Çok eski bile olsa herhangi bir önbellek verisi var mı?
	staleCacheExists := len(ws.validatorsCache) > 0
	staleValidators := ws.validatorsCache
	ws.validatorsCacheMutex.RUnlock()
	
	// Asenkron olarak validator listesini güncellemeye çalışalım
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in background validator update: %v", r)
			}
		}()
		
		// 5 saniyelik kısa bir timeout ile deneyelim
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Kanallarla iletişim kuralım
		done := make(chan bool, 1)
		var validators []blockchain.ValidatorInfo
		
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC in validator fetching: %v", r)
				}
				done <- true
			}()
			
			start := time.Now()
			log.Printf("Background fetching validators from blockchain")
			
			// Get validators from blockchain
			allValidators := ws.blockchain.GetValidators()
			log.Printf("Found %d total validators in blockchain", len(allValidators))
			
			// Filter only active validators
			validators = make([]blockchain.ValidatorInfo, 0)
			for _, v := range allValidators {
				// Check if validator is active by checking if they have mined any blocks
				hasMinedBlocks := false
				chainHeight := ws.blockchain.GetChainHeight()
				
				// Check last 10 blocks for this validator
				for i := uint64(0); i < 10 && i <= chainHeight; i++ {
					block, err := ws.blockchain.GetBlockByIndex(i)
					if err != nil {
						continue
					}
					if block.Validator == v.Address {
						hasMinedBlocks = true
						break
					}
				}
				
				if hasMinedBlocks {
					validators = append(validators, v)
					log.Printf("Found active validator: %s", v.Address)
				} else {
					log.Printf("Found inactive validator: %s", v.Address)
				}
			}
			
			if len(validators) > 0 {
				// Önbelleği güncelle
				ws.validatorsCacheMutex.Lock()
				ws.validatorsCache = validators
				ws.validatorsCacheTime = time.Now()
				ws.validatorsCacheMutex.Unlock()
				
				log.Printf("Background updated validator cache with %d active validators in %v", 
					len(validators), time.Since(start))
			} else {
				log.Printf("No active validators found in blockchain")
			}
		}()
		
		// Wait for either completion or timeout
		select {
		case <-done:
			// İşlem tamamlandı, önbellek güncellendi
		case <-ctx.Done():
			log.Printf("Background validator update timed out: %v", ctx.Err())
		}
	}()
	
	// Hemen yanıt verelim - Önce eski önbellek, yoksa varsayılan veri
	if staleCacheExists && len(staleValidators) > 0 {
		log.Printf("Returning %d validators from stale cache immediately", len(staleValidators))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(staleValidators)
		return
	}
	
	// Önbellekte hiç veri yoksa, boş liste döndür
	log.Printf("No validator cache available, returning empty list")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(defaultValidators)
}

// getConfirmedTransactions handles the confirmed transactions endpoint with caching
func (ws *WebServer) getConfirmedTransactions(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers and Content-Type
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Get limit parameter, default to 30
	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			// Cap the limit to prevent performance issues
			if l > 100 {
				l = 100
			}
			limit = l
		}
	}
	
	// Önbellekteki verileri kontrol edelim (15 saniyeden daha yeni ise)
	ws.confirmedTxCacheMutex.RLock()
	cacheAge := time.Since(ws.confirmedTxCacheTime)
	hasCache := len(ws.confirmedTxCache) > 0 && cacheAge < 15*time.Second
	
	// Eğer önbellekte güncel veri varsa ve istenen limit önbellek boyutundan az veya eşitse, hemen döndürelim
	if hasCache && limit <= len(ws.confirmedTxCache) {
		// Önbellekten limiti kadar veri alalım
		txs := ws.confirmedTxCache
		if limit < len(txs) {
			txs = txs[:limit]
		}
		ws.confirmedTxCacheMutex.RUnlock()
		
		log.Printf("Returning %d confirmed transactions from cache (age: %v)", len(txs), cacheAge)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(txs)
		return
	}
	ws.confirmedTxCacheMutex.RUnlock()
	
	// Önbellekte veri yoksa veya eski ise veya istenen limit önbellek boyutundan büyükse, yeni veri alalım
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second) // Kısa bir timeout kullanarak hızlı cevap dönelim
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	var confirmedTxs []*blockchain.Transaction
	var err error
	
	// Do the work in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in getConfirmedTransactions: %v", r)
				err = fmt.Errorf("internal error: %v", r)
			}
			done <- true
		}()
		
		start := time.Now()
		log.Printf("Getting confirmed transactions from blockchain, limit=%d", limit)
		
		// Initialize the result array
		confirmedTxs = make([]*blockchain.Transaction, 0)
		
		// Get blockchain height
		height := int(ws.blockchain.GetChainHeight())
		
		// Sadece son 10 bloğa bakalım
		maxBlocksToCheck := 10
		if height < maxBlocksToCheck {
			maxBlocksToCheck = height + 1
		}
		
		// En son limiti aşmamak için her bloktan az sayıda işlem alalım
		txsPerBlock := limit / maxBlocksToCheck
		if txsPerBlock < 5 {
			txsPerBlock = 5
		}
		
		// Onaylanmış işlemleri en son bloklardan alalım
		for i := height; i >= (height-maxBlocksToCheck+1) && i >= 0 && len(confirmedTxs) < limit; i-- {
			block, err := ws.blockchain.GetBlockByIndex(uint64(i))
			if err != nil {
				log.Printf("Error fetching block at index %d: %v", i, err)
				continue
			}
			
			// Her bloktan en son birkaç işlemi alalım
			txsToProcess := block.Transactions
			if len(txsToProcess) > txsPerBlock {
				txsToProcess = txsToProcess[len(txsToProcess)-txsPerBlock:]
			}
			
			for _, tx := range txsToProcess {
				// Skip coinbase/reward transactions
				if tx.From == "0" || tx.From == "" {
					continue
				}
				
				// Create a copy
				txCopy := *tx
				// Add status and block information
				txCopy.Status = "confirmed"
				txCopy.BlockIndex = int64(block.Index)
				txCopy.BlockHash = block.Hash
				
				confirmedTxs = append(confirmedTxs, &txCopy)
				
				if len(confirmedTxs) >= limit {
					break
				}
			}
		}
		
		// Önbelleği güncelle - tüm işlemleri saklayalım (limitle sınırlamadan)
		ws.confirmedTxCacheMutex.Lock()
		ws.confirmedTxCache = confirmedTxs
		ws.confirmedTxCacheTime = time.Now()
		ws.confirmedTxCacheMutex.Unlock()
		
		log.Printf("Retrieved %d confirmed transactions in %v", len(confirmedTxs), time.Since(start))
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		if err != nil {
			log.Printf("Error getting confirmed transactions: %v", err)
			// Return an empty array instead of an error
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(make([]*blockchain.Transaction, 0))
			return
		}
		
		// Create empty array if nil
		if confirmedTxs == nil {
			confirmedTxs = make([]*blockchain.Transaction, 0)
		}
		
		// Limit the response if needed
		if len(confirmedTxs) > limit {
			confirmedTxs = confirmedTxs[:limit]
		}
		
		// Return the transactions as JSON
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(confirmedTxs)
		if err != nil {
			log.Printf("Error encoding confirmed transactions: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
		
	case <-ctx.Done():
		log.Printf("Timeout getting confirmed transactions: %v", ctx.Err())
		
		// Önbellekte herhangi bir veri varsa, eski de olsa döndürelim
		ws.confirmedTxCacheMutex.RLock()
		hasCacheData := len(ws.confirmedTxCache) > 0
		cachedTxs := ws.confirmedTxCache // Kopyasını alalım
		if limit < len(cachedTxs) {
			cachedTxs = cachedTxs[:limit]
		}
		ws.confirmedTxCacheMutex.RUnlock()
		
		if hasCacheData {
			log.Printf("Returning %d confirmed transactions from stale cache due to timeout", len(cachedTxs))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(cachedTxs)
			return
		}
		
		// Hiç önbellek verisi yoksa boş dizi döndür
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(make([]*blockchain.Transaction, 0))
	}
}

// importWallet handles the wallet import endpoint
func (ws *WebServer) importWallet(w http.ResponseWriter, r *http.Request) {
	// Automatically handle CORS preflight request
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("Received wallet import request")

	var req struct {
		PrivateKey string `json:"privateKey"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	log.Printf("Request body decoded successfully, privateKey length: %d", len(req.PrivateKey))

	if req.PrivateKey == "" {
		log.Printf("Empty private key received")
		http.Error(w, "Private key is required", http.StatusBadRequest)
		return
	}

	// Import crypto/rand to use in this function
	privKey, err := blockchain.ImportPrivateKey(req.PrivateKey)
	if err != nil {
		log.Printf("Error importing private key: %v", err)
		http.Error(w, fmt.Sprintf("Invalid private key: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Private key imported successfully")

	// Use the private key from the blockchain package
	keyPair := &blockchain.KeyPair{
		PrivateKey: privKey,
		PublicKey:  &privKey.PublicKey,
	}

	// Generate address from public key
	address := blockchain.GenerateAddress(keyPair.PublicKey)

	// Check if wallet already exists in blockchain
	existingKeyPair, exists := ws.blockchain.GetKeyPair(address)
	if !exists {
		// Save wallet's key pair to blockchain
		ws.blockchain.AddKeyPair(address, keyPair)

		// Check if account exists, if not create it with initial balance
		_, err = ws.blockchain.GetBalance(address)
		if err != nil {
			initialBalance := big.NewInt(0) // Start with 0 tokens
			err = ws.blockchain.CreateAccount(address, initialBalance)
			if err != nil {
				log.Printf("Error creating account during import: %v", err)
			}
		}

		// Save blockchain state after import
		go ws.blockchain.SaveToDisk()
	} else {
		// Use existing key pair for consistent behavior
		keyPair = existingKeyPair
	}

	// Respond with wallet information
	response := struct {
		Address    string `json:"address"`
		PublicKey  string `json:"publicKey"`
		PrivateKey string `json:"privateKey"`
		Exists     bool   `json:"exists"`
	}{
		Address:    address,
		PublicKey:  keyPair.GetPublicKeyString(),
		PrivateKey: req.PrivateKey,
		Exists:     exists,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if !exists {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(response)
}

// Transfer handles the transfer endpoint
func (ws *WebServer) transfer(w http.ResponseWriter, r *http.Request) {
    log.Printf("Received transfer request from %s", r.RemoteAddr)
    
    // Set headers for CORS and content type
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

    // Handle preflight request
    if r.Method == "OPTIONS" {
        log.Printf("Handling OPTIONS request")
        w.WriteHeader(http.StatusOK)
        return
    }

    // Parse request body
    var req struct {
        From  string `json:"from"`
        To    string `json:"to"`
        Value string `json:"value"`
    }

    log.Printf("Attempting to parse request body")
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("Error parsing transfer request: %v", err)
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid request body",
        })
        return
    }
    log.Printf("Request parsed successfully: From=%s, To=%s, Value=%s", req.From, req.To, req.Value)

    // Convert value to uint64
    log.Printf("Converting value to uint64: %s", req.Value)
    value, err := strconv.ParseUint(req.Value, 10, 64)
    if err != nil {
        log.Printf("Error converting value to uint64: %v", err)
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid value format",
        })
        return
    }
    log.Printf("Value converted successfully: %d", value)

    // Validate request
    if req.From == "" || req.To == "" || value == 0 {
        log.Printf("Missing required fields: From=%s, To=%s, Value=%d", req.From, req.To, value)
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Missing required fields",
        })
        return
    }

    // Create transaction
    log.Printf("Creating transaction")
    simpleTransaction := &blockchain.Transaction{
        ID:        uuid.New().String(),
        From:      req.From,
        To:        req.To,
        Value:     value,
        Timestamp: time.Now().Unix(),
        Signature: []byte("system_transfer"),
        Type:      "regular",
        Status:    "pending",
    }
    log.Printf("Transaction created: ID=%s", simpleTransaction.ID)

    // Create a context with timeout for the transfer operation
    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()

    // Use a channel to get the result
    errCh := make(chan error, 1)

    go func() {
        log.Printf("Adding transaction to blockchain: %s", simpleTransaction.ID)
        // Add transaction to the blockchain
        err := ws.blockchain.AddTransaction(simpleTransaction)
        errCh <- err
    }()

    // Wait for the transaction to complete or timeout
    select {
    case err := <-errCh:
        if err != nil {
            log.Printf("Transfer error: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(map[string]interface{}{
                "error": fmt.Sprintf("Transfer failed: %v", err),
            })
            return
        }

        // Success - transaction was added to the pool
        log.Printf("Transaction added to pool: %s", simpleTransaction.ID)
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "success": true,
            "transaction": map[string]interface{}{
                "id":        simpleTransaction.ID,
                "from":      simpleTransaction.From,
                "to":        simpleTransaction.To,
                "value":     simpleTransaction.Value,
                "timestamp": simpleTransaction.Timestamp,
                "type":      simpleTransaction.Type,
                "status":    simpleTransaction.Status,
            },
        })

    case <-ctx.Done():
        log.Printf("Timeout creating transaction: %v", ctx.Err())
        w.WriteHeader(http.StatusGatewayTimeout)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Request timed out",
        })
    }
}

// getWalletBalanceSimple is a simplified version of getWalletBalance
func (ws *WebServer) getWalletBalanceSimple(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	// Get formatted balance
	balance, err := ws.blockchain.GetFormattedBalance(address)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get balance: %v", err), http.StatusInternalServerError)
		return
	}

	// Return just the balance as a string
	w.Write([]byte(balance))
}

// getHealthCheck provides a super fast health status endpoint for frontend connection checks
func (ws *WebServer) getHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Set headers for CORS
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Return a simple health status with no blockchain operations
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time": time.Now().Unix(),
	})
}

// Multi-signature request types
type CreateMultiSigWalletRequest struct {
	Address       string   `json:"address"`
	Owners        []string `json:"owners"`
	RequiredSigs  int      `json:"requiredSigs"`
	AdminAddress  string   `json:"adminAddress"`
	Signature     string   `json:"signature"`
}

type CreateMultiSigTransactionRequest struct {
	WalletAddress string `json:"walletAddress"`
	From          string `json:"from"`
	To            string `json:"to"`
	Value         string `json:"value"`
	Data          []byte `json:"data,omitempty"`
	Type          string `json:"type"`
	Signature     string `json:"signature"`
}

type SignMultiSigTransactionRequest struct {
	WalletAddress string `json:"walletAddress"`
	TxID          string `json:"txID"`
	Signer        string `json:"signer"`
	Signature     string `json:"signature"`
}

type ExecuteMultiSigTransactionRequest struct {
	WalletAddress string `json:"walletAddress"`
	TxID          string `json:"txID"`
	Signature     string `json:"signature"`
}

// Multi-signature handlers
func (ws *WebServer) createMultiSigWallet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req CreateMultiSigWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Create new multisig wallet
	wallet, err := blockchain.NewMultiSigWallet(uuid.New().String(), req.Owners, req.RequiredSigs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Add wallet to blockchain
	if err := ws.blockchain.CreateMultiSigWallet(wallet.Address, wallet.Owners, wallet.RequiredSigs); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"address": wallet.Address,
		"owners": wallet.Owners,
		"requiredSigs": wallet.RequiredSigs,
	})
}

func (ws *WebServer) getMultiSigWallet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	wallet, err := ws.blockchain.GetMultiSigWallet(address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response := struct {
		Address      string   `json:"address"`
		Owners       []string `json:"owners"`
		RequiredSigs int      `json:"requiredSigs"`
	}{
		Address:      wallet.Address,
		Owners:       wallet.Owners,
		RequiredSigs: wallet.RequiredSigs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) createMultiSigTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateMultiSigTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := ws.blockchain.CreateMultiSigTransaction(
		req.WalletAddress,
		req.From,
		req.To,
		req.Value,
		req.Data,
		req.Type,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tx)
}

func (ws *WebServer) signMultiSigTransaction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req SignMultiSigTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	wallet, err := ws.blockchain.GetMultiSigWallet(req.WalletAddress)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Wallet not found"})
		return
	}

	err = wallet.SignTransaction(req.TxID, req.Signer, req.Signature)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (ws *WebServer) executeMultiSigTransaction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req ExecuteMultiSigTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	wallet, err := ws.blockchain.GetMultiSigWallet(req.WalletAddress)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Wallet not found"})
		return
	}

	tx, err := wallet.ExecuteTransaction(req.TxID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(tx)
}

func (ws *WebServer) getMultiSigTransactionStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	vars := mux.Vars(r)
	walletAddress := vars["walletAddress"]
	txID := vars["txID"]

	wallet, err := ws.blockchain.GetMultiSigWallet(walletAddress)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Wallet not found"})
		return
	}

	status, err := wallet.GetTransactionStatus(txID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Transaction not found"})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": status})
}

// ... existing code ...

// revertTransaction reverts a transaction by its hash
func (ws *WebServer) revertTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hash"]

	// Verify admin signature
	var req types.SignedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	valid, err := ws.verifyAdminSignature(&req)
	if !valid || err != nil {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusUnauthorized)
		return
	}

	// Find and revert the transaction
	err = ws.blockchain.RevertTransaction(hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// ... existing code ...

// Stop gracefully shuts down the web server
func (ws *WebServer) Stop() error {
	if ws.server != nil {
		// Give the server 5 seconds to finish processing existing requests
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return ws.server.Shutdown(ctx)
	}
	return nil
}

// getBlockByIndex handles retrieving a specific block by its index
func (ws *WebServer) getBlockByIndex(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    vars := mux.Vars(r)
    indexStr := vars["index"]
    
    index, err := strconv.ParseUint(indexStr, 10, 64)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid block index",
        })
        return
    }
    
    block, err := ws.blockchain.GetBlockByIndex(index)
    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Block not found",
        })
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(block)
}

// getAllTransactions combines pending and confirmed transactions
func (ws *WebServer) getAllTransactions(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
				
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    // Get pending transactions
    pendingTxs := ws.blockchain.GetPendingTransactions()
    
    // Get confirmed transactions from recent blocks
    confirmedTxs := make([]*blockchain.Transaction, 0)
    height := ws.blockchain.GetChainHeight()
    
    // Only check last 10 blocks for performance
    startBlock := uint64(0)
    if height > 10 {
        startBlock = height - 10
    }
    
    for i := startBlock; i <= height; i++ {
        block, err := ws.blockchain.GetBlockByIndex(i)
        if err != nil {
            continue
        }
        confirmedTxs = append(confirmedTxs, block.Transactions...)
    }
    
    // Combine transactions
    allTxs := append(pendingTxs, confirmedTxs...)
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(allTxs)
}

// approveValidator handles approving a validator
func (ws *WebServer) approveValidator(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    var req struct {
        Address string `json:"address"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid request body",
        })
        return
    }
    
    if err := ws.validatorManager.ApproveValidator(req.Address, req.Address); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Validator approved successfully",
    })
}

// rejectValidator handles rejecting a validator
func (ws *WebServer) rejectValidator(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    var req struct {
        Address string `json:"address"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid request body",
        })
        return
    }
    
    if err := ws.validatorManager.RejectValidator(req.Address, req.Address, "rejected"); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Validator rejected successfully",
    })
}

// suspendValidator handles suspending a validator
func (ws *WebServer) suspendValidator(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    var req struct {
        Address string `json:"address"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid request body",
        })
        return
    }
    
    if err := ws.validatorManager.SuspendValidator(req.Address, req.Address, "suspended"); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Validator suspended successfully",
    })
}

// addAdmin handles adding a new admin
func (ws *WebServer) addAdmin(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
				
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    var req struct {
        Address string `json:"address"`
    }
				
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid request body",
        })
        return
    }
    
    if err := ws.validatorManager.AddAdmin(req.Address, req.Address); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Admin added successfully",
    })
}

// removeAdmin handles removing an admin
func (ws *WebServer) removeAdmin(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    var req struct {
        Address string `json:"address"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid request body",
        })
        return
    }
    
    if err := ws.validatorManager.RemoveAdmin(req.Address, req.Address); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Admin removed successfully",
    })
}

// listAdmins returns the list of current admins
func (ws *WebServer) listAdmins(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    admins := ws.validatorManager.GetAdmins()
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "admins": admins,
    })
}

// listProposals returns the list of governance proposals
func (ws *WebServer) listProposals(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    if ws.governance == nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Governance system not enabled",
        })
        return
    }
    
    proposals := ws.governance.ListProposals(consensus.ProposalStatusAll)
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "proposals": proposals,
    })
}

// getProposal returns details of a specific proposal
func (ws *WebServer) getProposal(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    if ws.governance == nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Governance system not enabled",
        })
        return
    }
    
    vars := mux.Vars(r)
    proposalID := vars["id"]
    
    proposal, err := ws.governance.GetProposal(proposalID)
    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Proposal not found",
        })
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(proposal)
}

// createProposal handles creating a new governance proposal
func (ws *WebServer) createProposal(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    if ws.governance == nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Governance system not enabled",
        })
        return
    }
    
    var req struct {
        Title       string `json:"title"`
        Description string `json:"description"`
        Type        consensus.ProposalType `json:"type"`
        Data        string `json:"data"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid request body",
        })
        return
    }
    
    proposal, err := ws.governance.CreateProposal(req.Title, req.Type, req.Data, req.Description)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(proposal)
}

// castVote handles casting a vote on a proposal
func (ws *WebServer) castVote(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    if ws.governance == nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Governance system not enabled",
        })
        return
    }
    
    var req struct {
        ProposalID string `json:"proposalId"`
        Voter      string `json:"voter"`
        Vote       bool   `json:"vote"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid request body",
        })
        return
    }
    
    err := ws.governance.CastVote(req.ProposalID, req.Voter, req.Vote)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Vote cast successfully",
    })
}

// verifyAdminSignature verifies the signature of an admin request
func (ws *WebServer) verifyAdminSignature(req *types.SignedRequest) (bool, error) {
    if req.AdminAddress == "" || req.Signature == "" {
        return false, fmt.Errorf("missing admin address or signature")
    }
    
    // Verify the admin exists
    admins := ws.validatorManager.GetAdmins()
    isAdmin := false
    for _, admin := range admins {
        if admin == req.AdminAddress {
            isAdmin = true
            break
        }
    }
    
    if !isAdmin {
        return false, fmt.Errorf("address is not an admin")
    }
    
    // Get admin's public key
    keyPair, exists := ws.blockchain.GetKeyPair(req.AdminAddress)
    if !exists {
        return false, fmt.Errorf("admin key pair not found")
    }
    
    // Verify signature
    dataBytes, err := json.Marshal(req.Data)
    if err != nil {
        return false, fmt.Errorf("invalid data format")
    }

    signatureBytes, err := hex.DecodeString(req.Signature)
    if err != nil {
        return false, fmt.Errorf("invalid signature format")
    }

    valid, err := keyPair.VerifySignature(dataBytes, signatureBytes)
    if err != nil {
        return false, fmt.Errorf("signature verification failed")
    }

    if !valid {
        return false, fmt.Errorf("invalid signature")
    }
    
    return true, nil
}

// CreateMultisigWallet handles the creation of a new multisig wallet
func (ws *WebServer) createMultisigWallet(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    var req struct {
        Owners         []string `json:"owners"`
        RequiredSigs   int      `json:"requiredSigs"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid request body",
        })
        return
    }
    
    // Create new multisig wallet
    wallet, err := blockchain.NewMultiSigWallet(uuid.New().String(), req.Owners, req.RequiredSigs)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    // Add wallet to blockchain
    if err := ws.blockchain.CreateMultiSigWallet(wallet.Address, wallet.Owners, wallet.RequiredSigs); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "address": wallet.Address,
        "owners": wallet.Owners,
        "requiredSigs": wallet.RequiredSigs,
    })
}

// SignMultisigTransaction handles signing a multisig transaction
func (ws *WebServer) signMultisigTransaction(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    var req struct {
        TxID      string `json:"txId"`
        Signer    string `json:"signer"`
        Signature string `json:"signature"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid request body",
        })
        return
    }
    
    // Get wallet from blockchain
    wallet, err := ws.blockchain.GetMultiSigWallet(req.TxID)
    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Wallet not found",
        })
        return
    }
    
    // Sign transaction
    err = wallet.SignTransaction(req.TxID, req.Signer, req.Signature)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Transaction signed successfully",
        "status": "pending",
    })
}

// ExecuteMultisigTransaction handles executing a multisig transaction
func (ws *WebServer) executeMultisigTransaction(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    var req struct {
        TxID string `json:"txId"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Invalid request body",
        })
        return
    }
    
    // Get wallet from blockchain
    wallet, err := ws.blockchain.GetMultiSigWallet(req.TxID)
    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Wallet not found",
        })
        return
    }
    
    // Execute transaction
    tx, err := wallet.ExecuteTransaction(req.TxID)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Transaction executed successfully",
        "transaction": tx,
    })
}

// GetMultisigTransactionStatus handles getting the status of a multisig transaction
func (ws *WebServer) getMultisigTransactionStatus(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    vars := mux.Vars(r)
    txID := vars["txId"]
    
    // Get wallet from blockchain
    wallet, err := ws.blockchain.GetMultiSigWallet(txID)
    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Wallet not found",
        })
        return
    }
    
    // Get transaction status
    status, err := wallet.GetTransactionStatus(txID)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": status,
    })
}

// Add the missing getMultiSigPendingTransactions method
func (ws *WebServer) getMultiSigPendingTransactions(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    
    vars := mux.Vars(r)
    walletAddress := vars["walletAddress"]

    wallet, err := ws.blockchain.GetMultiSigWallet(walletAddress)
    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]string{"error": "Wallet not found"})
        return
    }

    pendingTxs := wallet.GetPendingTransactions()
    json.NewEncoder(w).Encode(pendingTxs)
}