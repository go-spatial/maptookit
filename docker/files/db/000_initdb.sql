
CREATE DATABASE coord;

CREATE DATABASE grids;
\c grids
CREATE EXTENSION postgis;
CREATE EXTENSION postgis_topology;
CREATE SCHEMA grids;
CREATE ROLE osm LOGIN;
CREATE ROLE graticule LOGIN; 

GRANT USAGE ON SCHEMA grids to postgres;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA grids TO postgres;
SELECT PostGIS_version();




