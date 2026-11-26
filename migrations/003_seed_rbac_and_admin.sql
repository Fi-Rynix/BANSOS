INSERT INTO users (name, email, password, role) VALUES (
    'Admin BANSOS',
    'admin-bansos@gmail.com',
    '$2a$12$8tu2c8X1PK2FI9ACOX6gZ.paCF7KhP8GwNWO5wrSm2rYRJcklqOEm',
    'admin'
)
ON CONFLICT (email) DO NOTHING;
