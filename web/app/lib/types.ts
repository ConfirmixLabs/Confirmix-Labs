// Type definitions only

export interface BlockchainStatus {
    height: number;
    lastBlock: string;
}

export interface Block {
    Index: number;
    Timestamp: number;
    Transactions: Transaction[];
    Hash: string;
    PrevHash: string;
    Validator: string;
    Signature: string | null;
    Nonce: number;
    HumanProof: string;
}

export interface Transaction {
    ID: string;
    From: string;
    To: string;
    Value: number;
    Data: string;
    Timestamp: number;
    Signature: string;
    Type: string;
    Status?: string;
    BlockIndex?: number;
    BlockHash?: string;
}

export interface ValidatorInfo {
    address: string;
    humanProof: string;
}

export interface TransactionResult extends Transaction {
    blockMined?: boolean; // Flag to indicate if transaction was immediately mined
    status?: string;
    message?: string;
}

export interface Wallet {
    address: string;
    publicKey: string;
}

// API interface definition
export interface Api {
    getStatus(): Promise<BlockchainStatus>;
    getBlocks(): Promise<Block[]>;
    getTransactions(): Promise<Transaction[]>;
    getPendingTransactions(): Promise<Transaction[]>;
    getConfirmedTransactions(): Promise<Transaction[]>;
    createWallet(): Promise<{ address: string; publicKey: string }>;
    getBalance(address: string): Promise<number>;
    createTransaction(from: string, to: string, amount: number, data?: string): Promise<TransactionResult>;
    mineBlock(validator: string): Promise<Block>;
    registerValidator(data: { address: string; humanProof: string }): Promise<{ message: string; address: string }>;
    getValidators(): Promise<ValidatorInfo[]>;
} 