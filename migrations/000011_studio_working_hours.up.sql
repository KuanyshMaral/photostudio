-- Таблица для хранения рабочих часов студии (структурированный формат)
CREATE TABLE IF NOT EXISTS studio_working_hours (
                                                    id BIGSERIAL PRIMARY KEY,
                                                    studio_id BIGINT NOT NULL UNIQUE REFERENCES studios(id) ON DELETE CASCADE,
                                                    hours JSONB NOT NULL DEFAULT '[]'::jsonb,
                                                    created_at TIMESTAMPTZ DEFAULT NOW(),
                                                    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Индекс для быстрого поиска по studio_id
CREATE INDEX IF NOT EXISTS idx_studio_working_hours_studio ON studio_working_hours(studio_id);

-- ВСТАВЛЯЕМ ДЛЯ ВСЕХ СУЩЕСТВУЮЩИХ СТУДИЙ
DO $$
    DECLARE
        studio_record RECORD;
    BEGIN
        FOR studio_record IN SELECT id FROM studios LOOP
                INSERT INTO studio_working_hours (studio_id, hours) VALUES
                    (
                        studio_record.id,
                        '[
                          {"day_of_week": 0, "open_time": "00:00", "close_time": "00:00", "is_closed": true},
                          {"day_of_week": 1, "open_time": "10:00", "close_time": "20:00", "is_closed": false},
                          {"day_of_week": 2, "open_time": "10:00", "close_time": "20:00", "is_closed": false},
                          {"day_of_week": 3, "open_time": "10:00", "close_time": "20:00", "is_closed": false},
                          {"day_of_week": 4, "open_time": "10:00", "close_time": "20:00", "is_closed": false},
                          {"day_of_week": 5, "open_time": "10:00", "close_time": "20:00", "is_closed": false},
                          {"day_of_week": 6, "open_time": "12:00", "close_time": "18:00", "is_closed": false}
                        ]'
                    ) ON CONFLICT (studio_id) DO NOTHING;
            END LOOP;
    END $$;