DROP TABLE IF EXISTS twitter_users;
CREATE TABLE twitter_users (
  user_id BIGINT UNSIGNED NOT NULL,
  screen_name VARCHAR(15) NOT NULL,
  name VARCHAR(20) NOT NULL,
  lang VARCHAR(100) NOT NULL,
  updated_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (user_id)
) ENGINE InnoDB CHARSET utf8;

DROP TABLE IF EXISTS erase_tweets;
CREATE TABLE erase_tweets (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  twitter_tweet_id BIGINT UNSIGNED NOT NULL,
  tweet VARCHAR(140) NOT NULL,
  posted_at DATETIME NOT NULL,
  twitter_user_id BIGINT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  FOREIGN KEY (twitter_user_id) REFERENCES twitter_users (user_id)
) ENGINE InnoDB CHARSET utf8;

DROP TABLE IF EXISTS erase_errors;
CREATE TABLE erase_errors (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tried_twitter_user_id BIGINT UNSIGNED NOT NULL,
  twitter_tweet_id BIGINT UNSIGNED NOT NULL,
  status_code SMALLINT(3) UNSIGNED NOT NULL,
  error_message TEXT NOT NULL,
  updated_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id)
) ENGINE InnoDB CHARSET utf8;
