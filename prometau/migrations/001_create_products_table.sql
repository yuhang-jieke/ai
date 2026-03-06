-- Migration: Create products table
-- Date: 2026-03-03

CREATE TABLE IF NOT EXISTS products (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '商品 ID',
    name VARCHAR(255) NOT NULL COMMENT '商品名称',
    description TEXT COMMENT '商品描述',
    price DECIMAL(10, 2) NOT NULL DEFAULT 0.00 COMMENT '商品价格',
    stock INT NOT NULL DEFAULT 0 COMMENT '库存数量',
    category_id BIGINT DEFAULT 0 COMMENT '分类 ID',
    image_url VARCHAR(500) DEFAULT '' COMMENT '商品图片 URL',
    status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0=下架，1=上架',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_category_id (category_id),
    INDEX idx_status (status),
    INDEX idx_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品表';

-- Insert sample data
INSERT INTO products (name, description, price, stock, category_id, image_url, status) VALUES
('iPhone 15 Pro', 'Apple iPhone 15 Pro 256GB 钛金属', 8999.00, 100, 1, '/images/iphone15pro.jpg', 1),
('MacBook Pro 14', 'Apple MacBook Pro 14 寸 M3 芯片', 12999.00, 50, 2, '/images/macbookpro14.jpg', 1),
('AirPods Pro 2', 'Apple AirPods Pro 2 主动降噪', 1899.00, 200, 3, '/images/airpodspro2.jpg', 1),
('iPad Air', 'Apple iPad Air 10.9 寸 64GB', 4799.00, 80, 4, '/images/ipadair.jpg', 1),
('Apple Watch S9', 'Apple Watch Series 9 GPS 45mm', 3299.00, 120, 5, '/images/applewatches9.jpg', 1);
