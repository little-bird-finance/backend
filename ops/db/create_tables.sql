CREATE TABLE tb_expense (
    id VARCHAR(128) PRIMARY KEY,
    amount BIGINT,
    timestamp TIMESTAMP WITH TIME ZONE,
    place VARCHAR(255),
    who VARCHAR(255),
    what VARCHAR(255),
    createdAt TIMESTAMP WITH TIME ZONE,
    updatedAt TIMESTAMP WITH TIME ZONE
);
