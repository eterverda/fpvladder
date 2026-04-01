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
      return localStorage.getItem(STORAGE_KEY) || DEFAULT_THEME;
    } catch (e) {
      const match = document.cookie.match(new RegExp(STORAGE_KEY + "=([^;]+)"));
      return match ? match[1] : DEFAULT_THEME;
    }
  }

  function saveTheme(theme) {
    try {
      localStorage.setItem(STORAGE_KEY, theme);
    } catch (e) {
      document.cookie = STORAGE_KEY + "=" + theme + "; path=/; max-age=31536000";
    }
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
    return localStorage.getItem(CLASS_STORAGE_KEY);
  } catch (e) {
    const match = document.cookie.match(new RegExp(CLASS_STORAGE_KEY + "=([^;]+)"));
    return match ? match[1] : null;
  }
}

// Save class to storage
function saveClass(classValue) {
  try {
    localStorage.setItem(CLASS_STORAGE_KEY, classValue);
  } catch (e) {
    document.cookie = CLASS_STORAGE_KEY + "=" + classValue + "; path=/; max-age=31536000";
  }
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

    // Update active state in dropdown
    document.querySelectorAll(".menu-option").forEach((btn) => {
      btn.classList.toggle("active", btn.dataset.value === classValue);
    });
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

// Row click handler for data-href - just navigate, class is in storage
document.addEventListener("click", (e) => {
  const target = e.target.closest("[data-href]");
  if (target && !e.target.closest("a")) {
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
