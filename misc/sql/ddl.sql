DROP TABLE IF EXISTS erase_tweets;
CREATE TABLE erase_tweets (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  twitter_tweet_id BIGINT UNSIGNED NOT NULL,
  tweet VARCHAR(140) NOT NULL,
  posted_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id)
) ENGINE InnoDB CHARSET utf8;

DROP TABLE IF EXISTS erase_errors;
CREATE TABLE erase_errors (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  twitter_tweet_id BIGINT UNSIGNED NOT NULL,
  status_code SMALLINT(3) UNSIGNED NOT NULL,
  error_message TEXT NOT NULL,
  updated_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id)
) ENGINE InnoDB CHARSET utf8;
