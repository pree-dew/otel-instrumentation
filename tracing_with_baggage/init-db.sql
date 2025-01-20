-- user
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    email VARCHAR(100)
);

-- order
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER,
    amount DECIMAL
);
