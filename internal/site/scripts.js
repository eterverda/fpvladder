// Touch device detection
(function () {
  "use strict";
  if (window.matchMedia("(pointer: coarse)").matches || "ontouchstart" in window || navigator.maxTouchPoints > 0) {
    document.documentElement.classList.add("touch-device");
  }
})();

// Theme management
(function () {
  "use strict";

  const STORAGE_KEY = "fpvladder-theme";
  const DEFAULT_THEME = "light";

  // Calculate relative path from current page to index.html
  function getRelativePathToRoot() {
    const scripts = document.querySelectorAll('script[src*="scripts.js"]');
    for (const script of scripts) {
      const src = script.getAttribute("src");
      if (src) {
        const lastSlash = src.lastIndexOf("/");
        if (lastSlash >= 0) {
          return src.substring(0, lastSlash + 1);
        } else {
          return "./";
        }
      }
    }
    const path = window.location.pathname;
    const parts = path.split("/").filter((p) => p.length > 0);
    const lastPart = parts[parts.length - 1];
    if (lastPart && lastPart.includes(".")) {
      parts.pop();
    }
    if (parts.length > 0 && parts[0].includes(":")) {
      parts.shift();
    }
    const depth = parts.length;
    return depth === 0 ? "./" : "../".repeat(depth);
  }

  function getSavedTheme() {
    try {
      const theme = localStorage.getItem(STORAGE_KEY);
      if (theme) return theme;
    } catch (e) {
      // localStorage not available, try cookies
    }
    // Try cookies as fallback (for file:// protocol in Firefox)
    const match = document.cookie.match(new RegExp(STORAGE_KEY + "=([^;]+)"));
    return match ? match[1] : DEFAULT_THEME;
  }

  function saveTheme(theme) {
    // Save to both localStorage and cookies for cross-page compatibility
    try {
      localStorage.setItem(STORAGE_KEY, theme);
    } catch (e) {
      // Ignore
    }
    // Always save to cookies as fallback for file:// protocol
    document.cookie = STORAGE_KEY + "=" + theme + "; path=/; max-age=31536000";
  }

  function applyTheme(theme) {
    document.documentElement.setAttribute("data-theme", theme);
    updateThemeButtons(theme);
  }

  function updateThemeButtons(theme) {
    document.querySelectorAll(".menu-option").forEach((btn) => {
      if (btn.dataset.value === "light" || btn.dataset.value === "dark") {
        btn.classList.toggle("active", btn.dataset.value === theme);
      }
    });
  }

  function toggleDropdown() {
    const dropdown = document.getElementById("settingsDropdown");
    if (dropdown) {
      dropdown.classList.toggle("active");
    }
  }

  function closeDropdownOnClickOutside(e) {
    const settingsBtn = document.getElementById("settingsBtn");
    const settingsClose = document.getElementById("settingsClose");
    const dropdown = document.getElementById("settingsDropdown");

    if (
      dropdown &&
      !dropdown.contains(e.target) &&
      !settingsBtn.contains(e.target) &&
      !(settingsClose && settingsClose.contains(e.target))
    ) {
      dropdown.classList.remove("active");
    }
  }

  const savedTheme = getSavedTheme();
  applyTheme(savedTheme);

  function init() {
    document.addEventListener("DOMContentLoaded", function () {
      const backArrow = document.getElementById("backArrow");
      const rootPath = getRelativePathToRoot() + "index.html";

      if (backArrow) {
        const path = window.location.pathname;
        const isHomePage = document.body.dataset.hasBack === "false" || path === "/" || path.endsWith("/index.html");
        if (isHomePage) {
          backArrow.style.display = "none";
        } else {
          backArrow.style.display = "flex";
          backArrow.href = rootPath;
        }
      }

      const settingsBtn = document.getElementById("settingsBtn");
      if (settingsBtn) {
        settingsBtn.addEventListener("click", toggleDropdown);
      }

      const settingsClose = document.getElementById("settingsClose");
      if (settingsClose) {
        settingsClose.addEventListener("click", (e) => {
          e.stopPropagation();
          const dropdown = document.getElementById("settingsDropdown");
          if (dropdown) dropdown.classList.remove("active");
        });
      }

      document.addEventListener("click", closeDropdownOnClickOutside);

      document.querySelectorAll(".menu-option").forEach((btn) => {
        // Only theme options (light/dark)
        const value = btn.dataset.value;
        if (value !== "light" && value !== "dark") return;

        btn.addEventListener("click", function () {
          const theme = this.dataset.value;
          if (theme) {
            applyTheme(theme);
            saveTheme(theme);
            const dropdown = document.getElementById("settingsDropdown");
            if (dropdown) dropdown.classList.remove("active");
          }
        });
      });
    });
  }

  init();
})();

// Class management (global functions for cross-page persistence)
const CLASS_STORAGE_KEY = "fpvladder-class";

// Get saved class from storage
function getSavedClass() {
  try {
    const classValue = localStorage.getItem(CLASS_STORAGE_KEY);
    if (classValue) return classValue;
  } catch (e) {
    // localStorage not available, try cookies
  }
  // Try cookies as fallback (for file:// protocol in Firefox)
  const match = document.cookie.match(new RegExp(CLASS_STORAGE_KEY + "=([^;]+)"));
  return match ? match[1] : null;
}

// Save class to storage
function saveClass(classValue) {
  // Save to both localStorage and cookies for cross-page compatibility
  try {
    localStorage.setItem(CLASS_STORAGE_KEY, classValue);
  } catch (e) {
    // Ignore
  }
  // Always save to cookies as fallback for file:// protocol
  document.cookie = CLASS_STORAGE_KEY + "=" + classValue + "; path=/; max-age=31536000";
}

// Class management
(function () {
  "use strict";

  // Get available classes from page content
  function getAvailableClasses() {
    const classes = new Set();
    document.querySelectorAll(".class-content").forEach((el) => {
      if (el.dataset.class) {
        classes.add(el.dataset.class);
      }
    });
    return Array.from(classes);
  }

  // Show content for specific class, hide others
  function showClassContent(classValue) {
    // Show/hide content sections
    document.querySelectorAll(".class-content").forEach((el) => {
      if (el.dataset.class === classValue) {
        el.style.display = "block";

        // Hide future events section if no events for this class
        const futureSection = el.querySelector(".future-events-section");
        if (futureSection && futureSection.dataset.hasEvents === "false") {
          futureSection.style.display = "none";
        }
      } else {
        el.style.display = "none";
      }
    });

    // Update active state in dropdown (only class options, not theme)
    document.querySelectorAll(".menu-option").forEach((btn) => {
      const value = btn.dataset.value;
      if (value !== "light" && value !== "dark") {
        btn.classList.toggle("active", value === classValue);
      }
    });

    // Re-apply favorites filter for visible class only
    if (typeof window.applyFavoritesFilter === "function") {
      window.applyFavoritesFilter(classValue);
    }
  }

  // Initialize class switching
  function initClassSwitching() {
    const available = getAvailableClasses();

    // Hide class options that are not available on this page (skip theme options)
    let visibleClassCount = 0;
    document.querySelectorAll(".menu-option").forEach((btn) => {
      const value = btn.dataset.value;
      // Skip theme options
      if (value === "light" || value === "dark") return;

      if (value && !available.includes(value)) {
        btn.style.display = "none";
      } else {
        visibleClassCount++;
      }
    });

    // Hide class section title and divider if no classes available
    const classTitle = document.querySelector(".settings-title");
    if (classTitle && classTitle.textContent === "Класс" && visibleClassCount === 0) {
      classTitle.style.display = "none";
    }

    // Hide divider if no classes (divider is between class and theme sections)
    const divider = document.getElementById("classThemeDivider");
    if (divider && visibleClassCount === 0) {
      divider.style.display = "none";
    }

    if (available.length === 0) return;

    // Priority: saved class > first available
    const savedClass = getSavedClass();
    let targetClass = savedClass;

    // If saved class is not available, pick first available
    if (!targetClass || !available.includes(targetClass)) {
      targetClass = available[0];
    }

    // Show target class
    showClassContent(targetClass);

    // Save class (in case we picked first available)
    saveClass(targetClass);

    // Setup class option clicks
    document.querySelectorAll(".menu-option").forEach((btn) => {
      // Skip theme options (light/dark)
      if (btn.dataset.value === "light" || btn.dataset.value === "dark") return;

      btn.addEventListener("click", function () {
        const classValue = this.dataset.value;
        if (classValue && available.includes(classValue)) {
          showClassContent(classValue);
          saveClass(classValue);
          // Close dropdown
          const dropdown = document.getElementById("settingsDropdown");
          if (dropdown) dropdown.classList.remove("active");
        }
      });
    });
  }

  // Run on DOM ready
  document.addEventListener("DOMContentLoaded", initClassSwitching);
})();

// Favorites management
(function () {
  "use strict";

  const FAVORITES_KEY = "fpvladder-favorites";
  const FAVORITES_FILTER_KEY = "fpvladder-filters";

  // Three independent filters: heart, star, flag
  let activeFilters = {
    heart: false,
    star: false,
    flag: false,
  };

  function getActiveFilters() {
    try {
      const data = localStorage.getItem(FAVORITES_FILTER_KEY);
      if (data) return JSON.parse(data);
    } catch (e) {
      // Try cookies
    }
    try {
      const match = document.cookie.match(new RegExp(FAVORITES_FILTER_KEY + "=([^;]+)"));
      if (match) return JSON.parse(decodeURIComponent(match[1]));
    } catch (e) {
      // Ignore
    }
    return { heart: false, star: false, flag: false };
  }

  function saveActiveFilters(filters) {
    const data = JSON.stringify(filters);
    try {
      localStorage.setItem(FAVORITES_FILTER_KEY, data);
    } catch (e) {
      // Ignore
    }
    try {
      document.cookie = FAVORITES_FILTER_KEY + "=" + encodeURIComponent(data) + "; path=/; max-age=31536000";
    } catch (e) {
      // Ignore
    }
  }

  function getFavorites() {
    let data = null;
    try {
      data = localStorage.getItem(FAVORITES_KEY);
    } catch (e) {
      // localStorage not available, try cookies
    }
    // Try cookies as fallback (for file:// protocol)
    if (!data) {
      try {
        const match = document.cookie.match(new RegExp(FAVORITES_KEY + "=([^;]+)"));
        if (match) data = decodeURIComponent(match[1]);
      } catch (e) {
        // Ignore
      }
    }

    if (data) {
      try {
        const parsed = JSON.parse(data);
        // Simple check: if keys look like ids (contain "/"), use it
        const keys = Object.keys(parsed);
        if (keys.length > 0 && keys[0].includes("/")) {
          return parsed;
        }
      } catch (e) {
        // Ignore
      }
    }

    return {};
  }

  function saveFavorites(favorites) {
    const data = JSON.stringify(favorites);
    // Save to both localStorage and cookies for cross-page compatibility
    try {
      localStorage.setItem(FAVORITES_KEY, data);
    } catch (e) {
      // Ignore
    }
    // Always save to cookies as fallback for file:// protocol
    try {
      document.cookie = FAVORITES_KEY + "=" + encodeURIComponent(data) + "; path=/; max-age=31536000";
    } catch (e) {
      // Ignore
    }
  }

  function toggleFavorite(id, filterType) {
    const favorites = getFavorites();

    // Initialize array for this id if not exists
    if (!favorites[id]) {
      favorites[id] = [];
    }

    const index = favorites[id].indexOf(filterType);
    if (index === -1) {
      favorites[id].push(filterType);
    } else {
      favorites[id].splice(index, 1);
      // Remove empty arrays
      if (favorites[id].length === 0) {
        delete favorites[id];
      }
    }

    saveFavorites(favorites);
    updateSearchFiltersVisibility();
  }

  function isFavorite(id, filterType) {
    const favorites = getFavorites();
    return favorites[id] && favorites[id].includes(filterType);
  }

  function hasFavorites() {
    const favorites = getFavorites();
    return Object.keys(favorites).length > 0;
  }

  function updateFavoriteButtons() {
    document.querySelectorAll(".favorite-inline").forEach((el) => {
      const id = el.dataset.id;
      if (isFavorite(id, "heart")) {
        el.classList.add("active");
      } else {
        el.classList.remove("active");
      }
    });
  }

  function updateSearchFiltersVisibility() {
    const searchFilters = document.querySelectorAll(".search-filter");
    if (searchFilters.length === 0) return;

    // Don't show filters if search is active
    const searchInputs = document.querySelectorAll(".search-input");
    const isSearchActive = Array.from(searchInputs).some((input) => input.value.trim().length > 0);
    if (isSearchActive) {
      searchFilters.forEach((btn) => btn.classList.remove("visible"));
      return;
    }

    // Update visibility based on available favorites
    const favorites = getFavorites();

    searchFilters.forEach((filterBtn) => {
      const filterType = filterBtn.dataset.filter;
      const hasItems = Object.values(favorites).some((tags) => tags.includes(filterType));

      if (hasItems) {
        filterBtn.classList.add("visible");
      } else {
        filterBtn.classList.remove("visible");
        // Deactivate filter if no items
        if (activeFilters[filterType]) {
          activeFilters[filterType] = false;
          filterBtn.classList.remove("active");
        }
      }
    });

    saveActiveFilters(activeFilters);
  }

  function applyFavoritesFilter(visibleClass) {
    const favorites = getFavorites();

    // Check if search is active (has text)
    const searchInputs = document.querySelectorAll(".search-input");
    const isSearchActive = Array.from(searchInputs).some((input) => input.value.trim().length > 0);

    // If visibleClass specified, only filter rows in that class
    let rowsSelector = ".data-row[data-id]";
    if (visibleClass) {
      rowsSelector = `.class-content[data-class="${visibleClass}"] .data-row[data-id]`;
    }

    document.querySelectorAll(rowsSelector).forEach((row) => {
      const id = row.dataset.id;
      const itemTags = favorites[id] || [];

      // Check which filters are active
      const activeFilterTypes = [];
      if (activeFilters.heart) activeFilterTypes.push("heart");
      if (activeFilters.star) activeFilterTypes.push("star");
      if (activeFilters.flag) activeFilterTypes.push("flag");

      // If no filters active, show all (unless search is active)
      if (activeFilterTypes.length === 0) {
        if (!isSearchActive) {
          row.style.display = "";
        }
        return;
      }

      // Check if item has ALL active filters (AND logic)
      const hasAllActiveFilters = activeFilterTypes.every((filter) => itemTags.includes(filter));

      if (hasAllActiveFilters && !isSearchActive) {
        row.style.display = "";
      } else {
        row.style.display = "none";
      }
    });

    // Update empty states for visible class only
    let tablesSelector = ".pilot-table, .event-table";
    if (visibleClass) {
      tablesSelector = `.class-content[data-class="${visibleClass}"] .pilot-table, .class-content[data-class="${visibleClass}"] .event-table`;
    }

    document.querySelectorAll(tablesSelector).forEach((table) => {
      const tbody = table.querySelector("tbody");
      const emptyState = table.nextElementSibling;
      if (!tbody || !emptyState) return;

      const visibleRows = tbody.querySelectorAll('.data-row:not([style*="display: none"])');
      if (visibleRows.length === 0) {
        table.style.display = "none";
        emptyState.style.display = "block";
      } else {
        table.style.display = "table";
        emptyState.style.display = "none";
      }
    });
  }

  // Expose to global scope for class switching
  window.applyFavoritesFilter = applyFavoritesFilter;
  window.updateSearchFiltersVisibility = updateSearchFiltersVisibility;

  function initFavorites() {
    // Setup favorite inline icons (legacy)
    document.querySelectorAll(".favorite-inline").forEach((el) => {
      el.addEventListener("click", function (e) {
        e.preventDefault();
        e.stopPropagation();
        const id = this.dataset.id;
        const type = this.dataset.type;
        toggleFavorite(id, type);
      });
    });

    // Setup favorites widget (oval with 3 icons)
    document.querySelectorAll(".favorites-widget").forEach((widget) => {
      const id = widget.dataset.id;
      const row = widget.closest(".data-row, .event-row");

      // Heart icon
      const heart = widget.querySelector(".widget-icon.heart");
      if (heart) {
        heart.addEventListener("click", function (e) {
          e.preventDefault();
          e.stopPropagation();
          toggleFavorite(id, "heart");
          this.classList.toggle("active");
        });
      }

      // Star icon
      const star = widget.querySelector(".widget-icon.star");
      if (star) {
        star.addEventListener("click", function (e) {
          e.preventDefault();
          e.stopPropagation();
          toggleFavorite(id, "star");
          this.classList.toggle("active");
        });
      }

      // Flag icon
      const flag = widget.querySelector(".widget-icon.flag");
      if (flag) {
        flag.addEventListener("click", function (e) {
          e.preventDefault();
          e.stopPropagation();
          toggleFavorite(id, "flag");
          this.classList.toggle("active");
        });
      }
    });

    // Setup favorites filter
    const filterBtns = document.querySelectorAll(".favorites-filter");

    // Restore saved filter state
    // Initialize active filters from storage
    activeFilters = getActiveFilters();

    // Setup search filters (heart, star, flag)
    const searchFilters = document.querySelectorAll(".search-filter");
    searchFilters.forEach((filterBtn) => {
      const filterType = filterBtn.dataset.filter;

      // Show filter button if there are items with this tag
      const favorites = getFavorites();
      const hasItems = Object.values(favorites).some((tags) => tags.includes(filterType));
      if (hasItems) {
        filterBtn.classList.add("visible");
      }

      // Set initial active state
      if (activeFilters[filterType]) {
        filterBtn.classList.add("active");
      }

      filterBtn.addEventListener("click", function () {
        activeFilters[filterType] = !activeFilters[filterType];
        saveActiveFilters(activeFilters);
        this.classList.toggle("active", activeFilters[filterType]);
        applyFavoritesFilter();
      });
    });

    applyFavoritesFilter();

    // Initial update
    updateFavoriteButtons();
    updateSearchFiltersVisibility();
    updateFavoritesWidgets();
  }

  function updateFavoritesWidgets() {
    const favorites = getFavorites();

    document.querySelectorAll(".favorites-widget").forEach((widget) => {
      const id = widget.dataset.id;
      const tags = favorites[id] || [];

      // Update heart
      const heart = widget.querySelector(".widget-icon.heart");
      if (heart) {
        heart.classList.toggle("active", tags.includes("heart"));
      }

      // Update star
      const star = widget.querySelector(".widget-icon.star");
      if (star) {
        star.classList.toggle("active", tags.includes("star"));
      }

      // Update flag
      const flag = widget.querySelector(".widget-icon.flag");
      if (flag) {
        flag.classList.toggle("active", tags.includes("flag"));
      }
    });
  }

  // Run on DOM ready
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initFavorites);
  } else {
    initFavorites();
  }
})();

// Row click handler for data-href - just navigate, class is in storage
document.addEventListener("click", (e) => {
  const target = e.target.closest("[data-href]");
  if (
    target &&
    !e.target.closest("a") &&
    !e.target.closest(".favorite-inline") &&
    !e.target.closest(".favorites-widget")
  ) {
    window.location.href = target.dataset.href;
  }
});

// Smart header - hide on scroll down, show on scroll up
(function () {
  let lastScrollY = window.scrollY;
  let ticking = false;
  const header = document.querySelector(".site-header");
  if (!header) return;
  const headerHeight = 50;

  function updateHeader() {
    const currentScrollY = window.scrollY;

    if (currentScrollY <= 0) {
      header.style.transform = "translateY(0)";
    } else if (currentScrollY > lastScrollY && currentScrollY > headerHeight) {
      header.style.transform = "translateY(-100%)";
    } else if (currentScrollY < lastScrollY) {
      header.style.transform = "translateY(0)";
    }

    lastScrollY = currentScrollY;
    ticking = false;
  }

  window.addEventListener(
    "scroll",
    function () {
      if (!ticking) {
        window.requestAnimationFrame(updateHeader);
        ticking = true;
      }
    },
    { passive: true },
  );
})();
