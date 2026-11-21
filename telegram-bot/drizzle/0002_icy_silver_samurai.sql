CREATE TABLE "notification_deliveries" (
	"deduplication_key" text PRIMARY KEY NOT NULL,
	"notification_type" text NOT NULL,
	"telegram_id" bigint NOT NULL,
	"status" text DEFAULT 'claimed' NOT NULL,
	"claimed_at" timestamp with time zone DEFAULT now() NOT NULL,
	"sent_at" timestamp with time zone,
	"error_message" text
);
