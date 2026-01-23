CREATE TABLE comics (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    words TEXT[]
);