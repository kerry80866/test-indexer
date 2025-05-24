
CREATE TABLE progress_trade (
    slot        BIGINT PRIMARY KEY,     -- Solana slot
    source      SMALLINT NOT NULL,      -- 1=grpc, 2=rpc
    block_time  BIGINT NOT NULL,        -- Unix timestamp (秒或毫秒)
    status      SMALLINT NOT NULL DEFAULT 0, -- 0=未处理, 1=已处理, 2=slot无效
    updated_at  BIGINT NOT NULL         -- 写入时间（Unix 秒）
);

CREATE TABLE progress_balance (
      slot        BIGINT PRIMARY KEY,
      source      SMALLINT NOT NULL,
      block_time  BIGINT NOT NULL,
      status      SMALLINT NOT NULL DEFAULT 0,
      updated_at  BIGINT NOT NULL
);

CREATE INDEX idx_progress_trade_status_slot ON progress_trade(status, slot);
CREATE INDEX idx_progress_balance_status_slot ON progress_balance(status, slot);
