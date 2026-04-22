CREATE TABLE IF NOT EXISTS customer (
  id INT PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS purchase_order (
  id INT PRIMARY KEY,
  customer_id INT NOT NULL,
  amount DECIMAL(10,2) NOT NULL,
  status TEXT NOT NULL
);
INSERT INTO customer (id, name, email) VALUES (1, 'Gage Jacobson', 'minervaevans@farrell.com');
INSERT INTO customer (id, name, email) VALUES (2, 'Will Schmitt', 'berthahart@dooley.net');
INSERT INTO customer (id, name, email) VALUES (3, 'Chelsea Chapman', 'colbylee@glover.net');
INSERT INTO customer (id, name, email) VALUES (4, 'Courtney Beahan', 'carlievalencia@norman.info');
INSERT INTO customer (id, name, email) VALUES (5, 'Asa Nguyen', 'estellewu@koch.biz');
INSERT INTO customer (id, name, email) VALUES (6, 'Danny Glenn', 'dollyhammond@considine.name');
INSERT INTO customer (id, name, email) VALUES (7, 'Charles Morgan', 'otissummers@crawford.org');
INSERT INTO customer (id, name, email) VALUES (8, 'Julie Schaefer', 'nellieabshire@satterfield.name');
INSERT INTO customer (id, name, email) VALUES (9, 'Lorenzo Mayer', 'gileshernandez@ramirez.biz');
INSERT INTO customer (id, name, email) VALUES (10, 'Frank Harvey', 'aliacarter@lamb.info');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (1, 5, 418.21, 'shipped');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (2, 6, 64.41, 'shipped');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (3, 5, 138.76, 'shipped');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (4, 2, 499.49, 'delivered');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (5, 5, 87.37, 'shipped');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (6, 2, 485.16, 'delivered');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (7, 8, 415.36, 'delivered');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (8, 7, 353.29, 'shipped');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (9, 9, 213.39, 'pending');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (10, 6, 216.72, 'delivered');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (11, 9, 251.64, 'delivered');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (12, 7, 380.08, 'pending');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (13, 5, 201.25, 'delivered');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (14, 5, 338.41, 'shipped');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (15, 7, 231.6, 'shipped');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (16, 10, 160.18, 'delivered');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (17, 1, 78.84, 'delivered');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (18, 8, 164.26, 'delivered');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (19, 7, 494.49, 'pending');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (20, 6, 344.13, 'shipped');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (21, 8, 107.27, 'shipped');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (22, 5, 379.1, 'pending');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (23, 5, 90.37, 'pending');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (24, 9, 333.03, 'pending');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (25, 10, 72.55, 'shipped');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (26, 4, 378.46, 'pending');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (27, 8, 202.96, 'shipped');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (28, 9, 377.87, 'pending');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (29, 10, 35.87, 'pending');
INSERT INTO purchase_order (id, customer_id, amount, status) VALUES (30, 1, 72.23, 'shipped');
DELETE FROM purchase_order;
DELETE FROM customer;
DROP TABLE IF EXISTS purchase_order;
DROP TABLE IF EXISTS customer;
