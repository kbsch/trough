-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

-- Sources table
CREATE TABLE sources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    base_url TEXT NOT NULL,
    scraper_type TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Listings table
CREATE TABLE listings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id UUID REFERENCES sources(id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    url TEXT NOT NULL,

    -- Core fields
    title TEXT NOT NULL,
    description TEXT,
    asking_price BIGINT,
    revenue BIGINT,
    cash_flow BIGINT,
    ebitda BIGINT,
    inventory_value BIGINT,
    real_estate_included BOOLEAN DEFAULT false,
    real_estate_value BIGINT,

    -- Location
    city TEXT,
    state TEXT,
    zip_code TEXT,
    country TEXT DEFAULT 'US',
    lat DOUBLE PRECISION,
    lng DOUBLE PRECISION,

    -- Business details
    industry TEXT,
    industry_category TEXT,
    business_type TEXT,
    year_established INTEGER,
    employees INTEGER,
    reason_for_sale TEXT,

    -- Lease
    lease_expiration DATE,
    monthly_rent BIGINT,

    -- Franchise
    is_franchise BOOLEAN DEFAULT false,
    franchise_name TEXT,

    -- Raw data
    raw_data JSONB DEFAULT '{}',

    -- Metadata
    first_seen_at TIMESTAMPTZ DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,

    -- Search vector
    search_vector TSVECTOR,

    UNIQUE(source_id, external_id)
);

-- Scrape jobs table
CREATE TABLE scrape_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id UUID REFERENCES sources(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    listings_found INTEGER DEFAULT 0,
    listings_new INTEGER DEFAULT 0,
    listings_updated INTEGER DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_listings_search ON listings USING GIN(search_vector);
CREATE INDEX idx_listings_location ON listings(lat, lng) WHERE lat IS NOT NULL AND lng IS NOT NULL;
CREATE INDEX idx_listings_price ON listings(asking_price) WHERE is_active = true;
CREATE INDEX idx_listings_industry ON listings(industry) WHERE is_active = true;
CREATE INDEX idx_listings_state ON listings(state) WHERE is_active = true;
CREATE INDEX idx_listings_source ON listings(source_id);
CREATE INDEX idx_listings_active ON listings(is_active) WHERE is_active = true;
CREATE INDEX idx_listings_last_seen ON listings(last_seen_at);

CREATE INDEX idx_scrape_jobs_source ON scrape_jobs(source_id);
CREATE INDEX idx_scrape_jobs_status ON scrape_jobs(status);

-- Trigger to update search vector on insert/update
CREATE OR REPLACE FUNCTION listings_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := to_tsvector('english',
        COALESCE(NEW.title, '') || ' ' ||
        COALESCE(NEW.description, '') || ' ' ||
        COALESCE(NEW.industry, '') || ' ' ||
        COALESCE(NEW.city, '') || ' ' ||
        COALESCE(NEW.state, '')
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER listings_search_vector_trigger
    BEFORE INSERT OR UPDATE ON listings
    FOR EACH ROW
    EXECUTE FUNCTION listings_search_vector_update();
