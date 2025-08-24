-- Create role member type
CREATE TYPE room_role_type AS ENUM ('admin', 'member');

CREATE TABLE room_members (
    id BIGSERIAL PRIMARY KEY,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role room_role_type NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    left_at TIMESTAMP WITH TIME ZONE,
    last_read_msg_id TEXT, -- ObjectId (Mongo)
    unread_count INT DEFAULT 0,
    UNIQUE (room_id, user_id)
);

-- Index for finding all member in one room
CREATE INDEX idx_room_members_room_id ON room_members(room_id);

-- Index for finding all room that related in one user
CREATE INDEX idx_room_members_user_id ON room_members(user_id);