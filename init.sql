-- Sample database initialization for Docker demo
CREATE DATABASE IF NOT EXISTS myapp;
USE myapp;

-- Create sample tables
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_username (username),
    INDEX idx_email (email)
);

CREATE TABLE IF NOT EXISTS posts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    status ENUM('draft', 'published', 'archived') DEFAULT 'draft',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
);

CREATE TABLE IF NOT EXISTS comments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    post_id INT NOT NULL,
    user_id INT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_post_id (post_id),
    INDEX idx_user_id (user_id)
);

-- Insert sample data
INSERT INTO users (username, email, password_hash) VALUES
('admin', 'admin@example.com', '$2a$10$example_hash_1'),
('john_doe', 'john@example.com', '$2a$10$example_hash_2'),
('jane_smith', 'jane@example.com', '$2a$10$example_hash_3');

INSERT INTO posts (user_id, title, content, status) VALUES
(1, 'Welcome to DB-Taxi', 'This is a sample post to demonstrate the MySQL Web Explorer.', 'published'),
(2, 'Getting Started Guide', 'Learn how to use DB-Taxi to explore your MySQL databases.', 'published'),
(3, 'Advanced Features', 'Explore the advanced features of DB-Taxi.', 'draft');

INSERT INTO comments (post_id, user_id, content) VALUES
(1, 2, 'Great tool! Very helpful for database exploration.'),
(1, 3, 'I love the clean interface.'),
(2, 1, 'Thanks for the feedback!'),
(2, 3, 'The table structure view is excellent.');