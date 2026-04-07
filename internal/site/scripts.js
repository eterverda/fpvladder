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
          backArrow.style.visibility = "hidden";
        } else {
          backArrow.style.visibility = "visible";
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

  // Get class from URL hash (e.g., #75mm)
  function getClassFromHash() {
    const hash = window.location.hash;
    if (hash && hash.length > 1) {
      return hash.slice(1); // remove # prefix
    }
    return null;
  }

  // Update URL hash with class value
  function setClassHash(classValue) {
    if (classValue) {
      window.location.hash = classValue;
    } else {
      // Remove hash if no class
      history.replaceState(null, null, " ");
    }
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

    // Update active state in tabs
    document.querySelectorAll(".class-tab").forEach((btn) => {
      btn.classList.toggle("active", btn.dataset.class === classValue);
    });

    // Re-apply favorites filter for visible class only
    window.refreshFavoritesUI(classValue);
  }

  // Initialize class switching
  function initClassSwitching() {
    const available = getAvailableClasses();

    if (available.length === 0) return;

    // Priority: hash > saved class > first available
    const hashClass = getClassFromHash();
    const savedClass = getSavedClass();
    let targetClass = hashClass || savedClass;

    // If target class is not available, pick first available
    if (!targetClass || !available.includes(targetClass)) {
      targetClass = available[0];
    }

    // Show target class
    showClassContent(targetClass);

    // Update hash if it was different or empty
    if (hashClass !== targetClass) {
      setClassHash(targetClass);
    }

    // Save class (in case we picked first available or from hash)
    saveClass(targetClass);

    // Setup class tab clicks
    document.querySelectorAll(".class-tab").forEach((btn) => {
      btn.addEventListener("click", function () {
        const classValue = this.dataset.class;
        if (classValue && available.includes(classValue)) {
          showClassContent(classValue);
          saveClass(classValue);
          setClassHash(classValue);
        }
      });
    });

    // Setup navigation buttons (prev/next) - hide when no more classes
    const prevBtn = document.querySelector(".class-nav-prev");
    const nextBtn = document.querySelector(".class-nav-next");
    const prevLabel = document.querySelector(".class-nav-prev-label");
    const nextLabel = document.querySelector(".class-nav-next-label");

    // Mobile nav buttons
    const prevBtnMobile = document.querySelector(".class-nav-prev-mobile");
    const nextBtnMobile = document.querySelector(".class-nav-next-mobile");
    const prevLabelMobile = document.querySelector(".class-nav-prev-mobile-label");
    const nextLabelMobile = document.querySelector(".class-nav-next-mobile-label");

    // Map class values to display names
    const classNames = {};
    document.querySelectorAll(".class-content[data-class]").forEach((el) => {
      const classValue = el.dataset.class;
      const cardLabel = el.querySelector(".pilot-card-label span");
      if (cardLabel) {
        classNames[classValue] = cardLabel.textContent;
      }
    });

    function getCurrentIndex() {
      const hashClass = getClassFromHash();
      const currentClass = hashClass && available.includes(hashClass) ? hashClass : available[0];
      return available.indexOf(currentClass);
    }

    function updateNavButtons(currentIdx) {
      // Update prev buttons (desktop + mobile) - only update text if visible
      const hasPrev = currentIdx > 0;
      const prevLabelText = hasPrev ? classNames[available[currentIdx - 1]] || available[currentIdx - 1] : "";
      if (prevBtn && prevLabel) {
        prevBtn.style.visibility = hasPrev ? "visible" : "hidden";
        if (hasPrev) prevLabel.textContent = prevLabelText;
      }
      if (prevBtnMobile && prevLabelMobile) {
        prevBtnMobile.style.visibility = hasPrev ? "visible" : "hidden";
        if (hasPrev) prevLabelMobile.textContent = prevLabelText;
      }
      // Update next buttons (desktop + mobile) - only update text if visible
      const hasNext = currentIdx < available.length - 1;
      const nextLabelText = hasNext ? classNames[available[currentIdx + 1]] || available[currentIdx + 1] : "";
      if (nextBtn && nextLabel) {
        nextBtn.style.visibility = hasNext ? "visible" : "hidden";
        if (hasNext) nextLabel.textContent = nextLabelText;
      }
      if (nextBtnMobile && nextLabelMobile) {
        nextBtnMobile.style.visibility = hasNext ? "visible" : "hidden";
        if (hasNext) nextLabelMobile.textContent = nextLabelText;
      }
    }

    function handleNavClick(direction) {
      const currentIdx = getCurrentIndex();
      const newIdx = direction === "prev" ? currentIdx - 1 : currentIdx + 1;
      if (newIdx >= 0 && newIdx < available.length) {
        const newClass = available[newIdx];
        showClassContent(newClass);
        saveClass(newClass);
        setClassHash(newClass);
        // updateNavButtons will be called by hashchange event
      }
    }

    if (available.length > 1) {
      const currentIdx = getCurrentIndex();
      updateNavButtons(currentIdx);

      // Desktop buttons
      if (prevBtn) prevBtn.addEventListener("click", () => handleNavClick("prev"));
      if (nextBtn) nextBtn.addEventListener("click", () => handleNavClick("next"));

      // Mobile buttons
      if (prevBtnMobile) prevBtnMobile.addEventListener("click", () => handleNavClick("prev"));
      if (nextBtnMobile) nextBtnMobile.addEventListener("click", () => handleNavClick("next"));

      // Update buttons on hash change
      window.addEventListener("hashchange", function () {
        updateNavButtons(getCurrentIndex());
      });
    } else {
      // Hide nav buttons if only one class
      [prevBtn, nextBtn, prevBtnMobile, nextBtnMobile].forEach((btn) => {
        if (btn) btn.style.display = "none";
      });
    }

    // Listen for hash changes (back/forward buttons)
    window.addEventListener("hashchange", function () {
      const newClass = getClassFromHash();
      if (newClass && available.includes(newClass)) {
        showClassContent(newClass);
        saveClass(newClass);
      }
    });

    // Hide class tabs if only one class available
    const tabsContainer = document.querySelector(".class-tabs");
    if (tabsContainer && available.length <= 1) {
      tabsContainer.style.display = "none";
    }
  }

  // Run on DOM ready
  document.addEventListener("DOMContentLoaded", initClassSwitching);
})();

// Favorites management
(function () {
  "use strict";

  const FAVORITES_KEY = "fpvladder-favorites";
  const FAVORITES_FILTER_KEY = "fpvladder-filters";

  // Single active filter: heart, star, flag, or null
  let activeFilter = null;

  function getActiveFilter() {
    try {
      const data = localStorage.getItem(FAVORITES_FILTER_KEY);
      if (data) {
        const parsed = JSON.parse(data);
        if (typeof parsed === "string") return parsed;
        if (parsed && typeof parsed === "object") {
          if (parsed.heart) return "heart";
          if (parsed.star) return "star";
          if (parsed.flag) return "flag";
        }
      }
    } catch (e) {
      // Try cookies
    }
    try {
      const match = document.cookie.match(new RegExp(FAVORITES_FILTER_KEY + "=([^;]+)"));
      if (match) {
        const parsed = JSON.parse(decodeURIComponent(match[1]));
        if (typeof parsed === "string") return parsed;
        if (parsed && typeof parsed === "object") {
          if (parsed.heart) return "heart";
          if (parsed.star) return "star";
          if (parsed.flag) return "flag";
        }
      }
    } catch (e) {
      // Ignore
    }
    return null;
  }

  function saveActiveFilter(filter) {
    const data = JSON.stringify(filter);
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
      const isActive = activeFilter === filterType;

      if (hasItems || isActive) {
        filterBtn.classList.add("visible");
      } else {
        filterBtn.classList.remove("visible");
      }
    });
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

      // If no filter active, show all (unless search is active)
      if (!activeFilter) {
        if (!isSearchActive) {
          row.style.display = "";
        }
        return;
      }

      if (itemTags.includes(activeFilter) && !isSearchActive) {
        row.style.display = "";
      } else {
        row.style.display = "none";
      }
    });

    // Update empty states for visible class only
    let tablesSelector = ".pilot-table, .event-table, .future-event-table";
    if (visibleClass) {
      tablesSelector = `.class-content[data-class="${visibleClass}"] .pilot-table, .class-content[data-class="${visibleClass}"] .event-table, .class-content[data-class="${visibleClass}"] .future-event-table`;
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

  // Expose to global scope for class switching and search
  window.refreshFavoritesUI = refreshFavoritesUI;

  function initFavorites() {
    // Setup favorites widget (oval with 3 icons)
    document.querySelectorAll(".favorites-widget").forEach((widget) => {
      const id = widget.dataset.id;
      const row = widget.closest(".data-row");

      // Heart icon
      const heart = widget.querySelector(".widget-icon.heart");
      if (heart) {
        heart.addEventListener("click", function (e) {
          e.preventDefault();
          e.stopPropagation();
          toggleFavorite(id, "heart");
          this.classList.toggle("active");
          refreshFavoritesUI();
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
          refreshFavoritesUI();
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
          refreshFavoritesUI();
        });
      }
    });

    // Restore saved filter state
    activeFilter = getActiveFilter();

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
      if (activeFilter === filterType) {
        filterBtn.classList.add("active");
      }

      filterBtn.addEventListener("click", function () {
        const wasActive = activeFilter === filterType;
        // Deactivate all filters visually
        searchFilters.forEach((btn) => btn.classList.remove("active"));
        if (wasActive) {
          activeFilter = null;
        } else {
          activeFilter = filterType;
          this.classList.add("active");
        }
        saveActiveFilter(activeFilter);
        refreshFavoritesUI();
      });
    });

    refreshFavoritesUI();
  }

  function refreshFavoritesUI(visibleClass) {
    applyFavoritesFilter(visibleClass);
    updateSearchFiltersVisibility();

    const favorites = getFavorites();
    document.querySelectorAll(".favorites-widget, .favorites-tags").forEach((widget) => {
      const id = widget.dataset.id;
      const tags = favorites[id] || [];
      const heart = widget.querySelector(".widget-icon.heart, .tag-icon.heart");
      if (heart) heart.classList.toggle("active", tags.includes("heart"));
      const star = widget.querySelector(".widget-icon.star, .tag-icon.star");
      if (star) star.classList.toggle("active", tags.includes("star"));
      const flag = widget.querySelector(".widget-icon.flag, .tag-icon.flag");
      if (flag) flag.classList.toggle("active", tags.includes("flag"));
    });
  }

  // Run on DOM ready
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initFavorites);
  } else {
    initFavorites();
  }

  // Update widgets when page is restored from bfcache (back button)
  window.addEventListener("pageshow", (event) => {
    if (event.persisted) {
      refreshFavoritesUI();
    }
  });
})();

// Future event countdowns (computed client-side)
(function updateFutureEventCountdowns() {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());

  document.querySelectorAll(".future-event-countdown").forEach((cell) => {
    const dateStr = cell.dataset.date;
    if (!dateStr) return;

    const [year, month, day] = dateStr.split("-").map(Number);
    const eventDay = new Date(year, month - 1, day);
    const diffDays = Math.round((eventDay - today) / (1000 * 60 * 60 * 24));

    if (diffDays < 0) {
      cell.textContent = "было";
    } else if (diffDays < 7) {
      cell.textContent = diffDays + " дн.";
    } else if (diffDays < 28) {
      cell.textContent = Math.floor(diffDays / 7) + " нед.";
    } else {
      cell.textContent = Math.ceil(diffDays / 30) + " мес.";
    }
  });
})();

// Row click handler for data-href - just navigate, class is in storage
document.addEventListener("click", (e) => {
  const target = e.target.closest("[data-href]");
  if (
    target &&
    !e.target.closest("a") &&
    !e.target.closest(".favorites-widget") &&
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
