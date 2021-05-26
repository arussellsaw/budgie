
export interface Transaction {
    transaction_id: string;
    timestamp: Date;
    description: string;
    amount: number;
    currency: string;
}

export interface AccountNumber {
    iban: string;
    number: string;
    sort_code: string;
    swift_bic: string;
}

export interface Provider {
    display_name: string;
    logo_uri: string;
    logo_url: string;
    provider_id: string;
}

export interface Account {
    update_timestamp: Date;
    account_id: string;
    account_type: string;
    display_name: string;
    currency: string;
    account_number: AccountNumber;
    provider: Provider;
}

