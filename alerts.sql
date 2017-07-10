DROP DATABASE IF EXISTS crypto;
CREATE DATABASE crypto;

\c crypto;

CREATE TYPE time_delta_type AS ENUM ('1h', '24h', '7d');

--CREATE TABLE Coins (
--  name varchar(255) primary key
--);

CREATE TABLE CoinAlerts (
  ID SERIAL PRIMARY KEY,
  email varchar(255),
  coin varchar(255) references Coins (name),
  threshold_delta float,
  time_delta time_delta_type,
  notes varchar(1000), -- 1000 character limit in notes section.
  active boolean,
  created_at bigint
);

CREATE TABLE Notifications (
  ID SERIAL PRIMARY KEY,
  alert_id SERIAL references CoinAlerts (ID),
  email varchar(255),
  coin varchar(255) references coins (name),
  current_delta float, -- the actual delta change of the currency.
  threshold_delta float, -- the delta change threshold of the currency when the notification was generated.
  created_at bigint
);

--insert into Coins (name) values ('bitcoin');
--insert into Coins (name) values ('litecoin');
insert into CoinAlerts (email, coin, percent_delta, time_delta, notes, active) values
('chrisdistrict@gmail.com', 'bitcoin', -1.0, '7d', 'bitcoin alert notes', true);
insert into CoinAlerts (email, coin, percent_delta, time_delta, notes, active) values
('blackshoalgroup@gmail.com', 'bitcoin', .5, '1h', 'bitcoin alert notes', false);
insert into CoinAlerts (id, email, coin, percent_delta, time_delta, notes, active) values
(1, 'blackshoalgroup@gmail.com', 'ripple', .5, '1d', 'ripple alert notes', true);

insert into Notifications(id, alert_id, email, coin, percent_delta, threshold, time_triggered) values
(1, 1, 'blackshoalgroup@gmail.com', 'ripple', 1.5, .5, 1000);
