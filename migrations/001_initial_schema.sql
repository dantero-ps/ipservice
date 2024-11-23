CREATE TABLE ip_ranges (
    id SERIAL PRIMARY KEY,
    network CIDR NOT NULL,
    country_code CHAR(2) NOT NULL,
    ip_version INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_ip_ranges_network_unique ON ip_ranges (network);
CREATE INDEX idx_ip_ranges_network ON ip_ranges USING gist (network inet_ops);