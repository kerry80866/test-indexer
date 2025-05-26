
CREATE TABLE progress_slot (
    slot        BIGINT PRIMARY KEY,     -- Solana slot
    source      SMALLINT NOT NULL,      -- 1=grpc, 2=rpc
    block_time  BIGINT NOT NULL,        -- Unix timestamp (秒或毫秒)
    status      SMALLINT NOT NULL DEFAULT 0, -- 0=未处理, 1=已处理, 2=slot无效
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_progress_status_slot ON progress_slot(status, slot);
