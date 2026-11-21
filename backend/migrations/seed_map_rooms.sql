-- ============================================================================
-- SEED: All rooms from map SVG components
-- Generated from: ThirdFloor.tsx, SecondFloor.tsx, FirstFloor.tsx
-- 
-- Run with: podman exec ems-postgres psql -U postgres -d eventdb -f /path/to/seed_map_rooms.sql
-- Or via: cat seed_map_rooms.sql | podman exec -i ems-postgres psql -U postgres -d eventdb
-- ============================================================================

-- First, let's use UPSERT to avoid conflicts with existing data
-- This will insert new rooms or update existing ones

BEGIN;

-- ============================================================================
-- FIRST FLOOR (floor = 1)
-- ============================================================================

-- Special named locations on first floor
INSERT INTO rooms (id, name, room_number, block, floor, type, capacity, is_bookable, is_accessible)
VALUES
    ('LIBRARY', 'Library', '168', 'C1.1', 1, 'library', 200, false, true),
    ('MEDIKER', 'Mediker Medical Center', '169', 'C1.1', 1, 'medical', 10, false, true),
    ('DININGHALL', 'Dining Hall', '170', 'C1.3', 1, 'cafeteria', 300, false, true),
    ('GYM', 'Gymnasium', '171', 'C1.3', 1, 'gym', 100, true, true),
    ('ASSEMBLYHALL', 'Assembly Hall', '172', 'C1.1', 1, 'auditorium', 500, true, true),
    ('Media_Studio', 'Media Studio', '168', 'C1.1', 1, 'classroom', 20, true, false),
    ('Music_Club', 'Music Club', '155', 'C1.1', 1, 'common_area', 30, true, false),
    ('Psychological_Counseling_Center', 'Psychological Counseling Center', '167', 'C1.1', 1, 'office', 5, false, true)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    capacity = EXCLUDED.capacity,
    is_bookable = EXCLUDED.is_bookable,
    updated_at = NOW();

-- C1.1 Block - First Floor numbered rooms
INSERT INTO rooms (id, name, room_number, block, floor, type, is_bookable)
VALUES
    ('C1.1.221P', 'C1.1.221P', '221', 'C1.1', 1, 'classroom', true),
    ('C1.1.222P', 'C1.1.222P', '222', 'C1.1', 1, 'classroom', true),
    ('C1.1.223P', 'C1.1.223P', '223', 'C1.1', 1, 'classroom', true),
    ('C1.1.224P', 'C1.1.224P', '224', 'C1.1', 1, 'classroom', true),
    ('C1.1.225P', 'C1.1.225P', '225', 'C1.1', 1, 'classroom', true),
    ('C1.1.226P', 'C1.1.226P', '226', 'C1.1', 1, 'classroom', true),
    ('C1.1.227P', 'C1.1.227P', '227', 'C1.1', 1, 'classroom', true),
    ('C1.1.228P', 'C1.1.228P', '228', 'C1.1', 1, 'classroom', true),
    ('C1.1.229P', 'C1.1.229P', '229', 'C1.1', 1, 'classroom', true),
    ('C1.1.230P', 'C1.1.230P', '230', 'C1.1', 1, 'classroom', true),
    ('C1.1.231P', 'C1.1.231P', '231', 'C1.1', 1, 'classroom', true),
    ('C1.1.232P', 'C1.1.232P', '232', 'C1.1', 1, 'classroom', true),
    ('C1.1.233P', 'C1.1.233P', '233', 'C1.1', 1, 'classroom', true),
    ('C1.1.234P', 'C1.1.234P', '234', 'C1.1', 1, 'classroom', true),
    ('C1.1.235P', 'C1.1.235P', '235', 'C1.1', 1, 'classroom', true),
    ('C1.1.139', 'C1.1.139', '139', 'C1.1', 1, 'classroom', false),
    ('C1.1.140', 'C1.1.140', '140', 'C1.1', 1, 'classroom', false),
    ('C1.1.141', 'C1.1.141', '141', 'C1.1', 1, 'classroom', false),
    ('C1.1.142', 'C1.1.142', '142', 'C1.1', 1, 'classroom', false),
    ('C1.1.143', 'C1.1.143', '143', 'C1.1', 1, 'classroom', false),
    ('C1.1.156', 'C1.1.156', '156', 'C1.1', 1, 'classroom', false),
    ('C1.1.163', 'C1.1.163', '163', 'C1.1', 1, 'classroom', false),
    ('C1.1.164', 'C1.1.164', '164', 'C1.1', 1, 'classroom', false),
    ('C1.1.165', 'C1.1.165', '165', 'C1.1', 1, 'classroom', false)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    updated_at = NOW();

-- C1.2 Block - First Floor rooms
INSERT INTO rooms (id, name, room_number, block, floor, type, is_bookable)
VALUES
    ('C1.2.121K', 'C1.2.121K (Kaspersky Academy)', '121', 'C1.2', 1, 'computer_lab', true),
    ('C1.2.122K', 'C1.2.122K (Samsung Innovation Campus)', '122', 'C1.2', 1, 'computer_lab', true),
    ('C1.2.123K', 'C1.2.123K (Huawei)', '123', 'C1.2', 1, 'computer_lab', true),
    ('C1.2.124K', 'C1.2.124K (Cisco Networking Academy)', '124', 'C1.2', 1, 'computer_lab', true),
    ('C1.2.129', 'C1.2.129', '129', 'C1.2', 1, 'classroom', false),
    ('C1.2.133', 'C1.2.133', '133', 'C1.2', 1, 'classroom', false),
    ('C1.2.135', 'C1.2.135', '135', 'C1.2', 1, 'classroom', false),
    ('C1.2.136', 'C1.2.136', '136', 'C1.2', 1, 'classroom', false),
    ('C1.2.137P', 'C1.2.137P', '137', 'C1.2', 1, 'classroom', true),
    ('C1.2.138L', 'C1.2.138L', '138', 'C1.2', 1, 'lecture_hall', true),
    ('C1.2.139', 'C1.2.139', '139', 'C1.2', 1, 'classroom', false),
    ('C1.2.GARDEROB', 'Garderob', '140', 'C1.2', 1, 'other', false)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    updated_at = NOW();

-- C1.3 Block - First Floor rooms
INSERT INTO rooms (id, name, room_number, block, floor, type, is_bookable)
VALUES
    ('C1.3.121', 'C1.3.121', '121', 'C1.3', 1, 'classroom', false),
    ('C1.3.125', 'C1.3.125', '125', 'C1.3', 1, 'classroom', false),
    ('C1.3.126', 'C1.3.126', '126', 'C1.3', 1, 'classroom', false),
    ('C1.3.128', 'C1.3.128', '128', 'C1.3', 1, 'classroom', false),
    ('C1.3.129', 'C1.3.129', '129', 'C1.3', 1, 'classroom', false),
    ('C1.3.130', 'C1.3.130', '130', 'C1.3', 1, 'classroom', false),
    ('C1.3.168', 'C1.3.168 (Hygiene)', '168', 'C1.3', 1, 'other', false),
    ('C1.3.187', 'C1.3.187', '187', 'C1.3', 1, 'classroom', false),
    ('C1.3.188', 'C1.3.188', '188', 'C1.3', 1, 'classroom', false)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    updated_at = NOW();

-- ============================================================================
-- SECOND FLOOR (floor = 2)
-- ============================================================================

-- Special named locations on second floor
INSERT INTO rooms (id, name, room_number, block, floor, type, capacity, is_bookable)
VALUES
    ('AITUSA', 'Student Government (AITUSA)', '248', 'C1.2', 2, 'office', 20, true),
    ('Career_Center', 'Career Center', '272', 'C1.1', 2, 'office', 15, false),
    ('Registrar', 'Registrar Office', '270', 'C1.1', 2, 'office', 10, false)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    capacity = EXCLUDED.capacity,
    updated_at = NOW();

-- C1.1 Block - Second Floor rooms
INSERT INTO rooms (id, name, room_number, block, floor, type, is_bookable)
VALUES
    ('C1.1.238K', 'C1.1.238K', '238', 'C1.1', 2, 'computer_lab', true),
    ('C1.1.239K', 'C1.1.239K', '239', 'C1.1', 2, 'computer_lab', true),
    ('C1.1.240K', 'C1.1.240K', '240', 'C1.1', 2, 'computer_lab', true),
    ('C1.1.241K', 'C1.1.241K', '241', 'C1.1', 2, 'computer_lab', true),
    ('C1.1.242K', 'C1.1.242K', '242', 'C1.1', 2, 'computer_lab', true),
    ('C1.1.244K', 'C1.1.244K', '244', 'C1.1', 2, 'computer_lab', true),
    ('C1.1.245K', 'C1.1.245K', '245', 'C1.1', 2, 'computer_lab', true),
    ('C1.1.246', 'C1.1.246', '246', 'C1.1', 2, 'classroom', false),
    ('C1.1.248', 'C1.1.248', '248', 'C1.1', 2, 'classroom', false),
    ('C1.1.250', 'C1.1.250', '250', 'C1.1', 2, 'classroom', false),
    ('C1.1.251L', 'C1.1.251L', '251', 'C1.1', 2, 'lecture_hall', true),
    ('C1.1.252L', 'C1.1.252L', '252', 'C1.1', 2, 'lecture_hall', true),
    ('C1.1.253L', 'C1.1.253L', '253', 'C1.1', 2, 'lecture_hall', true),
    ('C1.1.254L', 'C1.1.254L', '254', 'C1.1', 2, 'lecture_hall', true),
    ('C1.1.255P', 'C1.1.255P', '255', 'C1.1', 2, 'classroom', true),
    ('C1.1.256P', 'C1.1.256P', '256', 'C1.1', 2, 'classroom', true),
    ('C1.1.260P', 'C1.1.260P', '260', 'C1.1', 2, 'classroom', true),
    ('C1.1.261', 'C1.1.261', '261', 'C1.1', 2, 'classroom', false),
    ('C1.1.262', 'C1.1.262', '262', 'C1.1', 2, 'classroom', false),
    ('C1.1.263', 'C1.1.263', '263', 'C1.1', 2, 'classroom', false),
    ('C1.1.264', 'C1.1.264', '264', 'C1.1', 2, 'classroom', false),
    ('C1.1.265', 'C1.1.265', '265', 'C1.1', 2, 'classroom', false),
    ('C1.1.266', 'C1.1.266', '266', 'C1.1', 2, 'classroom', false),
    ('C1.1.267', 'C1.1.267', '267', 'C1.1', 2, 'classroom', false),
    ('C1.1.268', 'C1.1.268', '268', 'C1.1', 2, 'classroom', false),
    ('C1.1.269', 'C1.1.269', '269', 'C1.1', 2, 'classroom', false),
    ('C1.1.270', 'C1.1.270 (Registrar Director)', '270', 'C1.1', 2, 'office', false),
    ('C1.1.271', 'C1.1.271 (Registrar Staff)', '271', 'C1.1', 2, 'office', false),
    ('C1.1.272', 'C1.1.272 (Career Center)', '272', 'C1.1', 2, 'office', false),
    ('C1.1.273', 'C1.1.273', '273', 'C1.1', 2, 'classroom', false)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    updated_at = NOW();

-- C1.2 Block - Second Floor rooms  
INSERT INTO rooms (id, name, room_number, block, floor, type, is_bookable)
VALUES
    ('C1.2.221P', 'C1.2.221P', '221', 'C1.2', 2, 'classroom', true),
    ('C1.2.222P', 'C1.2.222P', '222', 'C1.2', 2, 'classroom', true),
    ('C1.2.223K', 'C1.2.223K', '223', 'C1.2', 2, 'computer_lab', true),
    ('C1.2.224P', 'C1.2.224P', '224', 'C1.2', 2, 'classroom', true),
    ('C1.2.225P', 'C1.2.225P', '225', 'C1.2', 2, 'classroom', true),
    ('C1.2.226P', 'C1.2.226P', '226', 'C1.2', 2, 'classroom', true),
    ('C1.2.227P', 'C1.2.227P', '227', 'C1.2', 2, 'classroom', true),
    ('C1.2.228P', 'C1.2.228P', '228', 'C1.2', 2, 'classroom', true),
    ('C1.2.229P', 'C1.2.229P', '229', 'C1.2', 2, 'classroom', true),
    ('C1.2.230P', 'C1.2.230P', '230', 'C1.2', 2, 'classroom', true),
    ('C1.2.231K', 'C1.2.231K', '231', 'C1.2', 2, 'computer_lab', true),
    ('C1.2.232P', 'C1.2.232P', '232', 'C1.2', 2, 'classroom', true),
    ('C1.2.233L', 'C1.2.233L', '233', 'C1.2', 2, 'lecture_hall', true),
    ('C1.2.234K', 'C1.2.234K', '234', 'C1.2', 2, 'computer_lab', true),
    ('C1.2.237L', 'C1.2.237L', '237', 'C1.2', 2, 'lecture_hall', true),
    ('C1.2.239K', 'C1.2.239K', '239', 'C1.2', 2, 'computer_lab', true),
    ('C1.2.240K', 'C1.2.240K', '240', 'C1.2', 2, 'computer_lab', true),
    ('C1.2.241K', 'C1.2.241K', '241', 'C1.2', 2, 'computer_lab', true),
    ('C1.2.242K', 'C1.2.242K', '242', 'C1.2', 2, 'computer_lab', true),
    ('C1.2.243K', 'C1.2.243K', '243', 'C1.2', 2, 'computer_lab', true),
    ('C1.2.245', 'C1.2.245', '245', 'C1.2', 2, 'classroom', false),
    ('C1.2.246', 'C1.2.246', '246', 'C1.2', 2, 'classroom', false),
    ('C1.2.248L', 'C1.2.248L (AITUSA)', '248', 'C1.2', 2, 'office', true),
    ('C1.2.249L', 'C1.2.249L', '249', 'C1.2', 2, 'lecture_hall', true),
    ('C1.2.250L', 'C1.2.250L', '250', 'C1.2', 2, 'lecture_hall', true),
    ('C1.2.251L', 'C1.2.251L', '251', 'C1.2', 2, 'lecture_hall', true),
    ('C1.2.252K', 'C1.2.252K', '252', 'C1.2', 2, 'computer_lab', true),
    ('C1.2.254', 'C1.2.254', '254', 'C1.2', 2, 'classroom', false),
    ('C1.2.255', 'C1.2.255', '255', 'C1.2', 2, 'classroom', false)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    updated_at = NOW();

-- C1.3 Block - Second Floor rooms
INSERT INTO rooms (id, name, room_number, block, floor, type, is_bookable)
VALUES
    ('C1.3.221K', 'C1.3.221K', '221', 'C1.3', 2, 'computer_lab', true),
    ('C1.3.222P', 'C1.3.222P', '222', 'C1.3', 2, 'classroom', true),
    ('C1.3.223P', 'C1.3.223P', '223', 'C1.3', 2, 'classroom', true),
    ('C1.3.224P', 'C1.3.224P', '224', 'C1.3', 2, 'classroom', true),
    ('C1.3.225P', 'C1.3.225P', '225', 'C1.3', 2, 'classroom', true),
    ('C1.3.226P', 'C1.3.226P', '226', 'C1.3', 2, 'classroom', true),
    ('C1.3.227P', 'C1.3.227P', '227', 'C1.3', 2, 'classroom', true),
    ('C1.3.228K', 'C1.3.228K', '228', 'C1.3', 2, 'computer_lab', true),
    ('C1.3.229P', 'C1.3.229P', '229', 'C1.3', 2, 'classroom', true),
    ('C1.3.230P', 'C1.3.230P', '230', 'C1.3', 2, 'classroom', true),
    ('C1.3.231P', 'C1.3.231P', '231', 'C1.3', 2, 'classroom', true),
    ('C1.3.232P', 'C1.3.232P', '232', 'C1.3', 2, 'classroom', true),
    ('C1.3.233P', 'C1.3.233P', '233', 'C1.3', 2, 'classroom', true),
    ('C1.3.234K', 'C1.3.234K', '234', 'C1.3', 2, 'computer_lab', true),
    ('C1.3.235L', 'C1.3.235L', '235', 'C1.3', 2, 'lecture_hall', true),
    ('C1.3.236P', 'C1.3.236P', '236', 'C1.3', 2, 'classroom', true),
    ('C1.3.237P', 'C1.3.237P', '237', 'C1.3', 2, 'classroom', true),
    ('C1.3.240P', 'C1.3.240P', '240', 'C1.3', 2, 'classroom', true),
    ('C1.3.241P', 'C1.3.241P', '241', 'C1.3', 2, 'classroom', true),
    ('C1.3.243P', 'C1.3.243P', '243', 'C1.3', 2, 'classroom', true),
    ('C1.3.244P', 'C1.3.244P', '244', 'C1.3', 2, 'classroom', true),
    ('C1.3.246', 'C1.3.246', '246', 'C1.3', 2, 'classroom', false),
    ('C1.3.247P', 'C1.3.247P', '247', 'C1.3', 2, 'classroom', true),
    ('C1.3.248P', 'C1.3.248P', '248', 'C1.3', 2, 'classroom', true),
    ('C1.3.249K', 'C1.3.249K', '249', 'C1.3', 2, 'computer_lab', true),
    ('C1.3.250K', 'C1.3.250K', '250', 'C1.3', 2, 'computer_lab', true),
    ('C1.3.251K', 'C1.3.251K', '251', 'C1.3', 2, 'computer_lab', true),
    ('C1.3.252K', 'C1.3.252K', '252', 'C1.3', 2, 'computer_lab', true),
    ('C1.3.253K', 'C1.3.253K', '253', 'C1.3', 2, 'computer_lab', true),
    ('C1.3.254L', 'C1.3.254L', '254', 'C1.3', 2, 'lecture_hall', true),
    ('C1.3.255L', 'C1.3.255L', '255', 'C1.3', 2, 'lecture_hall', true),
    ('C1.3.257P', 'C1.3.257P', '257', 'C1.3', 2, 'classroom', true),
    ('C1.3.258P', 'C1.3.258P', '258', 'C1.3', 2, 'classroom', true),
    ('C1.3.259P', 'C1.3.259P', '259', 'C1.3', 2, 'classroom', true),
    ('C1.3.260P', 'C1.3.260P', '260', 'C1.3', 2, 'classroom', true),
    ('C1.3.261P', 'C1.3.261P', '261', 'C1.3', 2, 'classroom', true),
    ('C1.3.262P', 'C1.3.262P', '262', 'C1.3', 2, 'classroom', true),
    ('C1.3.263P', 'C1.3.263P', '263', 'C1.3', 2, 'classroom', true),
    ('C1.3.264L', 'C1.3.264L', '264', 'C1.3', 2, 'lecture_hall', true),
    ('C1.3.266P', 'C1.3.266P', '266', 'C1.3', 2, 'classroom', true),
    ('C1.3.267P', 'C1.3.267P', '267', 'C1.3', 2, 'classroom', true)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    updated_at = NOW();

-- ============================================================================
-- THIRD FLOOR (floor = 3)
-- ============================================================================

-- Special named locations on third floor
INSERT INTO rooms (id, name, room_number, block, floor, type, capacity, is_bookable)
VALUES
    ('FABLAB', 'FabLab Laboratory', '350', 'C1.3', 3, 'laboratory', 30, true),
    ('SE_DEPARTMENT', 'Software Engineering Department', '360', 'C1.3', 3, 'office', 20, false),
    ('DIRECTORS_OFFICE', 'Director Office', '359', 'C1.3', 3, 'office', 10, false),
    ('CREATIVE_INDUSTRIES', 'School of Creative Industries', '353', 'C1.3', 3, 'office', 15, false),
    ('FULLGYM', 'Full Gymnasium', '380', 'C1.3', 1, 'gym', 150, true),
    ('GYM_EQUIPMENT', 'Gym Equipment Area', '381', 'C1.3', 1, 'gym', 30, true),
    ('POLE', 'Pole Dance Studio', '382', 'C1.3', 1, 'gym', 15, true)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    capacity = EXCLUDED.capacity,
    updated_at = NOW();

-- C1.3 Block - Third Floor rooms (main classroom wing)
INSERT INTO rooms (id, name, room_number, block, floor, type, is_bookable)
VALUES
    -- VK labeled rooms (not bookable - technical/staff areas)
    ('C1.3.TIR', 'C1.3.TIR (Shooting Range)', 'TIR', 'C1.3', 3, 'other', false),
    -- Numbered classrooms
    ('C1.3.337', 'C1.3.337', '337', 'C1.3', 3, 'classroom', false),
    ('C1.3.338', 'C1.3.338', '338', 'C1.3', 3, 'classroom', false),
    ('C1.3.339', 'C1.3.339', '339', 'C1.3', 3, 'classroom', false),
    ('C1.3.340', 'C1.3.340', '340', 'C1.3', 3, 'classroom', false),
    ('C1.3.341', 'C1.3.341', '341', 'C1.3', 3, 'classroom', false),
    ('C1.3.342', 'C1.3.342', '342', 'C1.3', 3, 'classroom', false),
    ('C1.3.343', 'C1.3.343', '343', 'C1.3', 3, 'classroom', false),
    ('C1.3.344', 'C1.3.344', '344', 'C1.3', 3, 'classroom', false),
    ('C1.3.345', 'C1.3.345', '345', 'C1.3', 3, 'classroom', false),
    ('C1.3.346', 'C1.3.346', '346', 'C1.3', 3, 'classroom', false),
    -- Labs/Special rooms
    ('C1.3.353', 'C1.3.353 (Creative Industries)', '353', 'C1.3', 3, 'office', false),
    ('C1.3.354', 'C1.3.354', '354', 'C1.3', 3, 'classroom', false),
    ('C1.3.355', 'C1.3.355', '355', 'C1.3', 3, 'classroom', false),
    ('C1.3.356', 'C1.3.356', '356', 'C1.3', 3, 'classroom', false),
    ('C1.3.357P', 'C1.3.357P', '357', 'C1.3', 3, 'classroom', true),
    ('C1.3.358P', 'C1.3.358P', '358', 'C1.3', 3, 'classroom', true),
    ('C1.3.359P', 'C1.3.359P (Director Office)', '359', 'C1.3', 3, 'office', false),
    ('C1.3.360K', 'C1.3.360K (SE Department)', '360', 'C1.3', 3, 'office', false),
    ('C1.3.361', 'C1.3.361', '361', 'C1.3', 3, 'classroom', false),
    ('C1.3.362P', 'C1.3.362P', '362', 'C1.3', 3, 'classroom', true),
    ('C1.3.365L', 'C1.3.365L', '365', 'C1.3', 3, 'lecture_hall', true),
    ('C1.3.366L', 'C1.3.366L', '366', 'C1.3', 3, 'lecture_hall', true),
    ('C1.3.367K', 'C1.3.367K', '367', 'C1.3', 3, 'computer_lab', true),
    ('C1.3.370L', 'C1.3.370L', '370', 'C1.3', 3, 'lecture_hall', true)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    updated_at = NOW();

INSERT INTO rooms (id, name, room_number, block, floor, type, is_bookable)
VALUES
    ('C1.1.318K', 'C1.1.318K', '318', 'C1.1', 3, 'computer_lab', true),
    ('C1.1.320', 'C1.1.320', '320', 'C1.1', 3, 'classroom', false),
    ('C1.1.321', 'C1.1.321', '321', 'C1.1', 3, 'classroom', false),
    ('C1.1.322', 'C1.1.322', '322', 'C1.1', 3, 'classroom', false),
    ('C1.1.323', 'C1.1.323', '323', 'C1.1', 3, 'classroom', false),
    ('C1.1.324L', 'C1.1.324L', '324', 'C1.1', 3, 'lecture_hall', true),
    ('C1.1.325', 'C1.1.325', '325', 'C1.1', 3, 'classroom', false),
    ('C1.1.326L', 'C1.1.326L', '326', 'C1.1', 3, 'lecture_hall', true),
    ('C1.1.327', 'C1.1.327', '327', 'C1.1', 3, 'classroom', false),
    ('C1.1.328L', 'C1.1.328L', '328', 'C1.1', 3, 'lecture_hall', true),
    ('C1.1.329', 'C1.1.329', '329', 'C1.1', 3, 'classroom', false),
    ('C1.1.330', 'C1.1.330', '330', 'C1.1', 3, 'classroom', false),
    ('C1.1.332', 'C1.1.332', '332', 'C1.1', 3, 'classroom', false),
    ('C1.1.333', 'C1.1.333', '333', 'C1.1', 3, 'classroom', false),
    ('C1.1.334L', 'C1.1.334L', '334', 'C1.1', 3, 'lecture_hall', true),
    ('C1.1.335', 'C1.1.335', '335', 'C1.1', 3, 'classroom', false),
    ('C1.1.336', 'C1.1.336', '336', 'C1.1', 3, 'classroom', false),
    ('C1.1.337', 'C1.1.337', '337', 'C1.1', 3, 'classroom', false),
    ('C1.1.338', 'C1.1.338', '338', 'C1.1', 3, 'classroom', false),
    ('C1.1.341P', 'C1.1.341P', '341', 'C1.1', 3, 'classroom', true),
    ('C1.1.342P', 'C1.1.342P', '342', 'C1.1', 3, 'classroom', true),
    ('C1.1.343P', 'C1.1.343P', '343', 'C1.1', 3, 'classroom', true),
    ('C1.1.344P', 'C1.1.344P', '344', 'C1.1', 3, 'classroom', true),
    ('C1.1.346P', 'C1.1.346P', '346', 'C1.1', 3, 'classroom', true),
    ('C1.1.347P', 'C1.1.347P', '347', 'C1.1', 3, 'classroom', true),
    ('C1.1.348K', 'C1.1.348K', '348', 'C1.1', 3, 'computer_lab', true),
    ('C1.1.349P', 'C1.1.349P', '349', 'C1.1', 3, 'classroom', true),
    ('C1.1.350P', 'C1.1.350P', '350', 'C1.1', 3, 'classroom', true),
    ('C1.1.353P', 'C1.1.353P', '353', 'C1.1', 3, 'classroom', true),
    ('C1.1.354K', 'C1.1.354K', '354', 'C1.1', 3, 'computer_lab', true),
    ('C1.1.355P', 'C1.1.355P', '355', 'C1.1', 3, 'classroom', true),
    ('C1.1.357K', 'C1.1.357K', '357', 'C1.1', 3, 'computer_lab', true),
    ('C1.1.358K', 'C1.1.358K', '358', 'C1.1', 3, 'computer_lab', true),
    ('C1.1.360K', 'C1.1.360K', '360', 'C1.1', 3, 'computer_lab', true),
    ('C1.1.361K', 'C1.1.361K', '361', 'C1.1', 3, 'computer_lab', true),
    ('C1.1.365P', 'C1.1.365P', '365', 'C1.1', 3, 'classroom', true),
    ('C1.1.366P', 'C1.1.366P', '366', 'C1.1', 3, 'classroom', true)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    is_bookable = EXCLUDED.is_bookable,
    updated_at = NOW();

INSERT INTO rooms (id, name, room_number, block, floor, type, is_bookable)
VALUES
    ('C1.2.319', 'C1.2.319', '319', 'C1.2', 3, 'classroom', false),
    ('C1.2.320', 'C1.2.320', '320', 'C1.2', 3, 'classroom', false),
    ('C1.2.321', 'C1.2.321', '321', 'C1.2', 3, 'classroom', false),
    ('C1.2.323', 'C1.2.323', '323', 'C1.2', 3, 'classroom', false),
    ('C1.2.324', 'C1.2.324', '324', 'C1.2', 3, 'classroom', false),
    ('C1.2.325', 'C1.2.325', '325', 'C1.2', 3, 'classroom', false),
    ('C1.2.326', 'C1.2.326', '326', 'C1.2', 3, 'classroom', false),
    ('C1.2.327', 'C1.2.327', '327', 'C1.2', 3, 'classroom', false),
    ('C1.2.328', 'C1.2.328', '328', 'C1.2', 3, 'classroom', false),
    ('C1.2.329', 'C1.2.329', '329', 'C1.2', 3, 'classroom', false),
    ('C1.2.330', 'C1.2.330', '330', 'C1.2', 3, 'classroom', false),
    ('C1.2.331', 'C1.2.331', '331', 'C1.2', 3, 'classroom', false),
    ('C1.2.332', 'C1.2.332', '332', 'C1.2', 3, 'classroom', false),
    ('C1.2.334', 'C1.2.334', '334', 'C1.2', 3, 'classroom', false),
    ('C1.2.335', 'C1.2.335', '335', 'C1.2', 3, 'classroom', false),
    ('C1.2.336', 'C1.2.336', '336', 'C1.2', 3, 'classroom', false),
    ('C1.2.337', 'C1.2.337', '337', 'C1.2', 3, 'classroom', false),
    ('C1.2.338', 'C1.2.338', '338', 'C1.2', 3, 'classroom', false),
    ('C1.2.339', 'C1.2.339', '339', 'C1.2', 3, 'classroom', false),
    ('C1.2.340', 'C1.2.340', '340', 'C1.2', 3, 'classroom', false),
    ('C1.2.344', 'C1.2.344', '344', 'C1.2', 3, 'classroom', false),
    ('C1.2.346', 'C1.2.346', '346', 'C1.2', 3, 'classroom', false),
    ('C1.2.352', 'C1.2.352', '352', 'C1.2', 3, 'classroom', false),
    ('C1.2.354', 'C1.2.354', '354', 'C1.2', 3, 'classroom', false),
    ('C1.2.358', 'C1.2.358', '358', 'C1.2', 3, 'classroom', false),
    ('C1.2.359', 'C1.2.359', '359', 'C1.2', 3, 'classroom', false),
    ('C1.2.360', 'C1.2.360', '360', 'C1.2', 3, 'classroom', false),
    ('C1.2.362', 'C1.2.362', '362', 'C1.2', 3, 'classroom', false),
    ('C1.2.363', 'C1.2.363', '363', 'C1.2', 3, 'classroom', false),
    ('C1.2.364', 'C1.2.364', '364', 'C1.2', 3, 'classroom', false),
    ('C1.2.365', 'C1.2.365', '365', 'C1.2', 3, 'classroom', false),
    ('C1.2.366', 'C1.2.366', '366', 'C1.2', 3, 'classroom', false),
    ('C1.2.367', 'C1.2.367', '367', 'C1.2', 3, 'classroom', false),
    ('C1.2.368', 'C1.2.368', '368', 'C1.2', 3, 'classroom', false),
    ('C1.2.369', 'C1.2.369', '369', 'C1.2', 3, 'classroom', false),
    ('C1.2.370', 'C1.2.370', '370', 'C1.2', 3, 'classroom', false),
    ('C1.2.371', 'C1.2.371', '371', 'C1.2', 3, 'classroom', false),
    ('C1.2.374', 'C1.2.374', '374', 'C1.2', 3, 'classroom', false),
    ('C1.2.375', 'C1.2.375', '375', 'C1.2', 3, 'classroom', false),
    ('C1.2.376', 'C1.2.376', '376', 'C1.2', 3, 'classroom', false)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    is_bookable = EXCLUDED.is_bookable,
    updated_at = NOW();

INSERT INTO rooms (id, name, room_number, block, floor, type, is_bookable)
VALUES
    ('C1.3.318P', 'C1.3.318P', '318', 'C1.3', 3, 'classroom', true),
    ('C1.3.319P', 'C1.3.319P', '319', 'C1.3', 3, 'classroom', true),
    ('C1.3.321P', 'C1.3.321P', '321', 'C1.3', 3, 'classroom', true),
    ('C1.3.322P', 'C1.3.322P', '322', 'C1.3', 3, 'classroom', true),
    ('C1.3.323K', 'C1.3.323K', '323', 'C1.3', 3, 'computer_lab', true),
    ('C1.3.324K', 'C1.3.324K', '324', 'C1.3', 3, 'computer_lab', true),
    ('C1.3.327K', 'C1.3.327K', '327', 'C1.3', 3, 'computer_lab', true),
    ('C1.3.328', 'C1.3.328', '328', 'C1.3', 3, 'classroom', false),
    ('C1.3.331P', 'C1.3.331P', '331', 'C1.3', 3, 'classroom', true),
    ('C1.3.352', 'C1.3.352', '352', 'C1.3', 3, 'classroom', false)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    floor = EXCLUDED.floor,
    type = EXCLUDED.type,
    is_bookable = EXCLUDED.is_bookable,
    updated_at = NOW();

COMMIT;

-- ============================================================================
-- Summary Query - Run after to verify
-- ============================================================================
-- SELECT block, floor, COUNT(*) as room_count 
-- FROM rooms 
-- GROUP BY block, floor 
-- ORDER BY block, floor;
