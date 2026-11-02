// MapLibre GL JS 6.16.0 (static/vendor/maplibre).
// The style URL and attribution are data attributes rendered from config
// (MAP_STYLE_URL and MAP_ATTRIBUTION). This file does not name a provider.

import * as maplibregl from "/static/vendor/maplibre/maplibre-gl.mjs";

// Cary, North Carolina. Used until a trail collection arrives.
const fallbackCenter = [-78.78, 35.79];

const pine = "#1d4a38";
const clay = "#c4623a";
const paper = "#fffaf3";

function flagged(value) {
  return value === "yes" || value === true || value === "true";
}

function boundsOf(features) {
  let bounds = null;
  const visit = (coord) => {
    if (!Array.isArray(coord) || typeof coord[0] !== "number" || typeof coord[1] !== "number") {
      return;
    }
    if (!bounds) {
      bounds = new maplibregl.LngLatBounds(coord, coord);
      return;
    }
    bounds.extend(coord);
  };
  const walk = (geometry) => {
    if (!geometry || !geometry.coordinates) {
      return;
    }
    if (geometry.type === "Point") {
      visit(geometry.coordinates);
    } else if (geometry.type === "LineString" || geometry.type === "MultiPoint") {
      for (const coord of geometry.coordinates) {
        visit(coord);
      }
    } else if (geometry.type === "MultiLineString" || geometry.type === "Polygon") {
      for (const line of geometry.coordinates) {
        for (const coord of line) {
          visit(coord);
        }
      }
    }
  };
  for (const feature of features) {
    walk(feature.geometry);
  }
  return bounds;
}

function trailheads(data) {
  const features = [];
  for (const feature of data.features || []) {
    const props = feature.properties || {};
    if (typeof props.trailhead_lng !== "number" || typeof props.trailhead_lat !== "number") {
      continue;
    }
    features.push({
      type: "Feature",
      geometry: {
        type: "Point",
        coordinates: [props.trailhead_lng, props.trailhead_lat],
      },
      properties: props,
    });
  }
  return { type: "FeatureCollection", features };
}

function popupNode(props) {
  const node = document.createElement("div");
  node.className = "map-popup";
  const title = document.createElement("strong");
  title.textContent = props.name || "Trail";
  const meta = document.createElement("p");
  meta.textContent = [props.distance, props.difficulty, props.dog, props.park]
    .filter((part) => typeof part === "string" && part !== "")
    .join(" · ");
  const link = document.createElement("a");
  const href = typeof props.href === "string" && props.href.startsWith("/trails/") ? props.href : "/trails";
  link.href = href;
  link.textContent = "View trail";
  node.append(title, meta, link);
  if (flagged(props.logged)) {
    const logged = document.createElement("p");
    logged.className = "meta";
    logged.textContent = "Logged";
    node.append(logged);
  }
  return node;
}

function setStatus(el, message) {
  const status = el.parentElement && el.parentElement.querySelector(".map-status");
  if (status) {
    status.textContent = message;
  }
}

async function mount(el) {
  const styleURL = el.dataset.styleUrl;
  if (!styleURL) {
    setStatus(el, "The map style is not configured.");
    return;
  }
  const map = new maplibregl.Map({
    container: el,
    style: styleURL,
    center: fallbackCenter,
    zoom: 9,
    attributionControl: false,
  });
  map.addControl(new maplibregl.NavigationControl({ showCompass: false }), "top-right");
  // The style's own TileJSON attribution stays in the map corner. The
  // caption under the map is the configured credit (MAP_ATTRIBUTION), so
  // it is not added here a second time.
  map.addControl(new maplibregl.AttributionControl({ compact: false }));

  map.on("load", async () => {
    let data;
    try {
      const response = await fetch(el.dataset.geojson, {
        headers: { Accept: "application/geo+json" },
      });
      if (!response.ok) {
        throw new Error(String(response.status));
      }
      data = await response.json();
    } catch (err) {
      setStatus(el, "Trail lines could not be loaded.");
      console.error(err);
      return;
    }

    map.addSource("trails", { type: "geojson", data });
    map.addSource("trailheads", { type: "geojson", data: trailheads(data) });
    map.addLayer({
      id: "trails-halo",
      type: "line",
      source: "trails",
      layout: { "line-cap": "round", "line-join": "round" },
      paint: { "line-color": paper, "line-width": 10 },
    });
    map.addLayer({
      id: "trails",
      type: "line",
      source: "trails",
      layout: { "line-cap": "round", "line-join": "round" },
      paint: {
        "line-color": ["case", ["==", ["get", "logged"], "yes"], clay, pine],
        "line-width": [
          "case",
          ["==", ["get", "selected"], "yes"],
          8,
          ["==", ["get", "logged"], "yes"],
          7,
          5,
        ],
      },
    });
    map.addLayer({
      id: "trailheads",
      type: "circle",
      source: "trailheads",
      paint: {
        "circle-radius": [
          "case",
          ["==", ["get", "logged"], "yes"],
          10,
          ["==", ["get", "selected"], "yes"],
          8,
          6,
        ],
        "circle-color": ["case", ["==", ["get", "logged"], "yes"], clay, pine],
        "circle-stroke-width": 2,
        "circle-stroke-color": paper,
      },
    });

    const bounds = boundsOf(data.features || []);
    if (bounds) {
      map.fitBounds(bounds, { padding: 48, maxZoom: 13, duration: 0 });
    }

    const popup = new maplibregl.Popup({ offset: 12, maxWidth: "260px" });
    const openPopup = (event) => {
      const box = [
        [event.point.x - 10, event.point.y - 10],
        [event.point.x + 10, event.point.y + 10],
      ];
      const features = map.queryRenderedFeatures(box, { layers: ["trails", "trailheads"] });
      if (!features.length) {
        return;
      }
      popup.setLngLat(event.lngLat).setDOMContent(popupNode(features[0].properties || {})).addTo(map);
    };
    map.on("click", openPopup);
    for (const id of ["trails", "trailheads"]) {
      map.on("mouseenter", id, () => {
        map.getCanvas().style.cursor = "pointer";
      });
      map.on("mouseleave", id, () => {
        map.getCanvas().style.cursor = "";
      });
    }
  });
}

for (const el of document.querySelectorAll("[data-trail-map]")) {
  mount(el);
}
