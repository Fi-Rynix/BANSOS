INSERT INTO users (name, email, password, role) VALUES (
    'Admin BANSOS',
    'admin-bansos@gmail.com',
    '$2a$12$LjQsdM.GdjePl5hRJjo1feSyTToQgzgXIRdpnkCOEtHNN2DolicwK',
    'admin'
)
ON CONFLICT (email) DO NOTHING;
