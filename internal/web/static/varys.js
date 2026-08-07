const TOKEN_PREFIX = "token:";
const HOME_PATH = "/feeds/all";
const DESKTOP_MEDIA_QUERY = "(min-width: 1280px)";

function is_desktop_mode() {
    return window.matchMedia(DESKTOP_MEDIA_QUERY).matches;
}

function load_desktop_assets() {
    if (!is_desktop_mode()) return;

    let stylesheet = document.createElement("link");
    stylesheet.rel = "stylesheet";
    stylesheet.href = "/static/varys-desktop.css";
    let start_mobile = () => {
        if (stylesheet.parentNode) {
            stylesheet.parentNode.removeChild(stylesheet);
        }
        start_mobile_mode();
    };
    stylesheet.onerror = start_mobile;
    stylesheet.onload = () => {
        let script = document.createElement("script");
        script.src = "/static/varys-desktop.js";
        script.onerror = start_mobile;
        script.onload = () => desktop_start().catch(start_mobile);
        document.head.append(script);
    };
    document.head.append(stylesheet);
}

load_desktop_assets();

function start_mode_detection() {
    let desktop_query = window.matchMedia(DESKTOP_MEDIA_QUERY);
    let reload = () => window.location.reload();
    if (desktop_query.addEventListener) {
        desktop_query.addEventListener("change", reload);
    } else {
        desktop_query.addListener(reload);
    }
}

function get_current_state() {
    let url = window.location.pathname;
    let hash = window.location.hash;

    if (hash) {
        hash = hash.substr(1);
        return {
            login: hash.startsWith(TOKEN_PREFIX),
            token: hash.substr(TOKEN_PREFIX.length),
            parts: hash.split("/").slice(1)
        }
    }

    return {
        login: false,
        token: "",
        parts: url.split("/").slice(1)
    }
}

async function fetch_json(url) {
    let rsp = await fetch(url);
    let data = await rsp.json();
    return [rsp, data];
}

async function fetch_feeds() {
    return fetch_json("/api/feeds");
}

async function fetch_feed(uid) {
    return fetch_json(`/api/feeds/${uid}`);
}

async function fetch_item(fuid, iuid) {
    if (fuid === "all") {
        let [feed_response, feed] = await fetch_feed(fuid);
        if (feed_response.status !== 200) {
            return [feed_response, feed];
        }

        let item = feed.items?.find(item => item.uid === iuid);
        if (item) {
            fuid = item.feed_uid;
        }
    }

    return fetch_json(`/api/feeds/${fuid}/items/${iuid}`);
}

async function login(token) {
    return fetch("/login", {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({token})
    });
}

async function mark_feed_as_read(fuid, before) {
    let rsp = await fetch(`/api/feeds/${fuid}/read`, {
        method: "POST",
        body: JSON.stringify({before}),
        headers: {"Content-Type": "application/json"},
    });
    return rsp.status === 200;
}

async function mark_item_as_read(fuid, iuid) {
    let rsp = await fetch(`/api/feeds/${fuid}/items/${iuid}/read`, {method: "POST"});
    return rsp.status === 200;
}

async function mark_feed_as_read_and_refresh(fuid, before) {
    set_loading();
    await mark_feed_as_read(fuid, before);
    refresh();
}

async function mark_all_feeds_as_read_and_refresh(feed) {
    set_loading();
    await mark_feed_as_read(feed.uid, feed.last_updated);
    refresh();
}

function create_element(tag, {class_name, text, children = []} = {}) {
    let element = document.createElement(tag);
    if (class_name) element.className = class_name;
    if (text !== undefined) element.textContent = text;
    element.append(...children);
    return element;
}

function replace_children(element, ...children) {
    while (element.firstChild) {
        element.removeChild(element.firstChild);
    }
    element.append(...children);
}

function table_row(label, value) {
    return create_element("tr", {
        children: [
            create_element("td", {text: label}),
            create_element("td", {children: [value]}),
        ],
    });
}

function gen_feed_list(feeds, opts = {}) {
    let feed_list = create_element("ol", {class_name: "feed-list"});
    let feed_fragment = document.createDocumentFragment();
    feeds.forEach(feed => {
        let a = create_element("a", {text: feed.name});
        let feed_url = opts.feed_url
            ? opts.feed_url(feed)
            : `/feeds/${feed.uid}`;
        link(a, feed_url, opts.link_handler);

        let unread = feed.item_count - feed.read_count;
        if (unread) {
            a.append(create_element("span", {
                class_name: "feed-unread-count",
                text: unread,
            }));
        }

        let class_name = unread ? "" : "feed-read";
        if (feed.uid === opts.selected_uid) {
            class_name += " feed-selected";
        }
        let feed_item = create_element("li", {
            class_name,
            children: [a],
        });
        feed_item.dataset.feedUid = feed.uid;
        feed_fragment.append(feed_item);
    });

    if (opts.include_status !== false) {
        let status_link = create_element("a", {
            class_name: "feed-status",
            text: "Status",
        });
        link(status_link, "/feeds/status", opts.link_handler);
        feed_fragment.append(create_element("li", {
            children: [status_link],
        }));
    }

    feed_list.append(feed_fragment);
    return feed_list;
}

function gen_item_list(feed, opts = {}) {
    let item_list = create_element("div", {class_name: "item-list"});
    if (!feed.items?.length) {
        item_list.append(create_element("div", {
            class_name: "empty-message",
            text: "No items.",
        }));
        return item_list;
    }

    let item_fragment = document.createDocumentFragment();
    feed.items.forEach(item => {
        let item_url = opts.item_url
            ? opts.item_url(item)
            : `/feeds/${feed.uid}/items/${item.uid}`;
        item_fragment.append(gen_item(item, {
            list_view: true,
            item_url,
            link_handler: opts.link_handler,
            selected: item.uid === opts.selected_uid,
        }));
    });
    item_list.append(item_fragment);
    return item_list;
}

function gen_status_content(feeds, opts = {}) {
    let container = create_element("div");
    let feed_fragment = document.createDocumentFragment();
    const THIRTY_DAYS = 30 * 24 * 60 * 60;
    feeds.forEach(feed => {
        if (feed.uid === "all") return;

        let status = "";
        if (feed.last_error && feed.last_error.trim() !== "") {
            status = "🔴";
        } else if (feed.last_updated && (
            (now() - feed.last_item) > THIRTY_DAYS ||
            feed.item_count === 0) ||
            feed.last_item === 0) {
            status = "🟡";
        } else {
            status = "🟢";
        }

        let a = create_element("a", {text: status + " " + feed.name});
        link(a, `/feeds/${feed.uid}`, opts.link_handler);

        let table = create_element("table", {class_name: "feed-status-table"});
        table.append(
            table_row("URL", create_element("code", {text: feed.url || ""})),
            table_row("Items", `${feed.item_count} total, ${feed.item_count - feed.read_count} unread`),
            table_row("Last update", feed.last_updated ? relative_time_desc(feed.last_updated) : ""),
        );

        if (feed.last_item) {
            table.append(table_row("Last item", relative_time_desc(feed.last_item)));
        }

        if (feed.last_error) {
            table.append(table_row("Error", feed.last_error ? feed.last_error : "none"));
        }

        feed_fragment.append(create_element("div", {
            class_name: "feed-status-block",
            children: [
                create_element("div", {
                    class_name: "feed-status-name",
                    children: [a],
                }),
                table,
            ],
        }));
    });

    container.append(feed_fragment);
    return container;
}

// Page state.
function error(err) {
    console.trace(err);
    set_content(create_element("div", {class_name: "error", text: err}));
    reset_controls({breadcrumbs: true});
}

async function show_feeds() {
    let rsp, data;
    try {
        reset_controls({breadcrumbs: true, read_button: true});
        set_loading();
        [rsp, data] = await fetch_feeds();
    } catch (err) {
        error(`Unexpected error fetching feed list: ${err}`);
        return;
    }

    switch (rsp.status) {
        case 401:
            error("Please login.");
            break;
        case 500:
            error(`Unexpected error: ${data.message}`);
            break;
        case 200:
            let all_feed = data.find(feed => feed.uid === "all");
            reset_controls({
                breadcrumbs: true,
                read_button: () => mark_all_feeds_as_read_and_refresh(all_feed),
            });
            set_content(gen_feed_list(data));
            break;
        default:
            error(`Unexpected response code: ${rsp.status}`);
            break;
    }
}

async function show_feed(uid) {
    let rsp, data;
    try {
        reset_controls({breadcrumbs: true, read_button: true});
        set_loading();
        [rsp, data] = await fetch_feed(uid);
    } catch (err) {
        error(`Unexpected error fetching feed: ${err}`);
        return;
    }

    switch (rsp.status) {
        case 401:
            error("Please login.");
            break;
        case 404:
            error("Feed not found.");
            break;
        case 500:
            error(`Unexpected error: ${data.message}`);
            break;
        case 200:
            reset_controls({
                breadcrumbs: {uid: data.uid, name: data.name},
                read_button: () => mark_feed_as_read_and_refresh(data.uid, data.last_updated),
            });
            set_content(gen_item_list(data));
            break;
        default:
            error(`Unexpected response code: ${rsp.status}`);
            break;
    }
}

async function show_item(fuid, iuid) {
    let rsp, data;

    try {
        set_loading();
        [rsp, data] = await fetch_item(fuid, iuid);
    } catch (err) {
        error(`Unexpected error fetching item: ${err}`);
        return;
    }

    switch (rsp.status) {
        case 401:
            error("Please login.");
            break;
        case 404:
            error("Item not found.");
            break;
        case 500:
            error(`Unexpected error: ${data.message}`);
            break;
        case 200:
            reset_controls({
                breadcrumbs: {
                    uid: fuid,
                    name: fuid === "all" ? "All" : data.feed_name,
                },
                open_button: data.url,
            });
            set_content(gen_item(data));
            window.scrollTo(0, 0);
            if (!data.read) {
                mark_item_as_read(data.feed_uid, data.uid);
            }
            break;
        default:
            error(`Unexpected response code: ${rsp.status}`);
            break;
    }
}

function gen_item_header(item, opts = {}) {
    let list_view = opts.list_view || false;
    let details = [];
    details.push(item.feed_name);

    if (item.authors) {
        let authors = item.authors.length > 32
            ? item.authors.substr(0, 32) + "..."
            : item.authors;
        details.push("by " + authors);
    }

    let when = relative_time_desc(item.timestamp);
    details.push(when);

    return create_element("div", {
        class_name: "item-header",
        children: [
            create_element("div", {
                class_name: list_view ? "item-title" : "item-title item-title-bold",
                text: item.title,
            }),
            create_element("div", {
                class_name: "item-details",
                text: details.join(" · "),
            }),
        ],
    });
}

function gen_item_content(item) {
    let content_div = create_element("div", {class_name: "item-content"});
    content_div.innerHTML = item.content;
    return content_div;
}

function gen_item(item, opts) {
    opts = opts || {};
    let list_view = (opts && opts.list_view) || false;

    let header_div = gen_item_header(item, {list_view});
    let item_children = [header_div];
    if (!list_view) {
        item_children.push(gen_item_content(item));
    }

    let item_div = create_element("div", {
        class_name: list_view ? "item-summary" : "item-full",
        children: item_children,
    });

    if (list_view) {
        let class_name = item.read ? "item-link item-link-read" : "item-link";
        if (opts.selected) {
            class_name += " item-link-selected";
        }
        let item_link = create_element("a", {
            class_name,
            children: [item_div],
        });
        item_link.dataset.itemUid = item.uid;
        link(item_link, opts.item_url || `/feeds/${item.feed_uid}/items/${item.uid}`, opts.link_handler);
        return item_link;
    }

    return item_div;
}

async function refresh() {
    let state = get_current_state();
    if (state.login) {
        let rsp = await login(state.token);
        if (rsp.status === 200) {
            history.replaceState(null, "", HOME_PATH);
            refresh();
        } else {
            error("Please login.");
        }
        return;
    }

    if (state.parts[0] === "feeds" && state.parts[2] === "items") {
        await show_item(state.parts[1], state.parts[3]);
    } else if (state.parts[0] === "feeds" && state.parts[1] === "status") {
        await show_status_feeds();
    } else if (state.parts[0] === "feeds") {
        if (state.parts[1]) {
            await show_feed(state.parts[1]);
        } else {
            await show_feeds();
        }
    } else {
        error("Please login.");
    }
}

async function show_status_feeds() {
    let rsp, data;
    set_loading();
    try {
        [rsp, data] = await fetch_feeds();
    } catch (err) {
        error(`Unexpected error fetching feed list: ${err}`);
        return;
    }

    switch (rsp.status) {
        case 401:
            error("Please login.");
            break;
        case 500:
            error(`Unexpected error: ${data.message}`);
            break;
        case 200:
            reset_controls({breadcrumbs: true});
            set_content(gen_status_content(data));
            break;
        default:
            error(`Unexpected response code: ${rsp.status}`);
            break;
    }
}

function set_content(...content) {
    replace_children(document.querySelector("#content"), ...content);
}

function set_loading() {
    let div = create_element("div", {
        class_name: "loading",
        children: [create_element("div", {class_name: "spinner"})],
    });
    // iOS flashes an enlarged scrollbar in the loading screen when the
    // scrollbar is still being displayed because of a recent scroll event. The
    // following code makes us overlay the loading screen on iOS to prevent
    // that from happening.
    let isIOS = /iPad|iPhone|iPod/.test(navigator.platform);
    if (isIOS) {
        div.classList.add("loading-ios");
        let elem = document.querySelector("#content");
        window.scrollTo(0, 0);
        elem.childNodes.forEach(node => {
            node.classList.add("invisible");
        })
        elem.append(div);
    } else {
        set_content(div);
    }
}

function save_scroll_position() {
    history.replaceState({
        ...history.state,
        scroll: {x: window.scrollX, y: window.scrollY},
    }, "");
}

function restore_scroll_position(state) {
    if (state?.scroll) {
        window.scrollTo(state.scroll.x, state.scroll.y);
    }
}

function link(a, url, navigate = refresh) {
    a.setAttribute("href", url);
    if (a.onclick) return;
    a.onclick = e => {
        save_scroll_position();
        history.pushState(null, "", a.href);
        navigate();
        e.preventDefault();
    };
}

function reset_controls(config) {
    if (!config) {
        document.querySelector("#controls").classList.add("hidden");
        return;
    }

    document.querySelector("#controls").classList.remove("hidden");

    let breadcrumbs = document.querySelector("#breadcrumbs");
    breadcrumbs.classList.add("hidden");
    if (config.breadcrumbs) {
        breadcrumbs.classList.remove("hidden");

        let items = document.querySelector("#breadcrumb-items");
        replace_children(items);

        let feeds_link = create_element("a", {text: "Feeds"});
        link(feeds_link, "/feeds");
        items.append(create_element("div", {
            class_name: "breadcrumb-item",
            children: [feeds_link],
        }));

        if (typeof config.breadcrumbs === 'object' && config.breadcrumbs !== null) {
            let feed_link = create_element("a", {text: config.breadcrumbs.name});
            link(feed_link, `/feeds/${config.breadcrumbs.uid}`);
            items.append(create_element("li", {
                class_name: "breadcrumb-item",
                children: [feed_link],
            }));
        }
    }

    let read_button = document.querySelector("#read-button");
    read_button.classList.add("hidden");
    read_button.onclick = null;
    if (config.read_button) {
        read_button.classList.remove("hidden");
        if (typeof config.read_button === "function") {
            read_button.onclick = config.read_button;
        }
    }

    let open_button = document.querySelector("#open-button");
    open_button.classList.add("hidden");
    if (config.open_button) {
        link(open_button, config.open_button);
        open_button.setAttribute("target", "_blank");
        open_button.setAttribute("rel", "noopener noreferrer");
        open_button.classList.remove("hidden");
    }
}

function start_mobile_mode() {
    if (document.readyState === "loading") {
        window.addEventListener("load", start_mobile_mode, {once: true});
        return;
    }

    start_mode_detection();

    let read_button = document.querySelector("#read-button");
    read_button.addEventListener("touchstart", () => { });

    let open_button = document.querySelector("#open-button");
    open_button.addEventListener("touchstart", () => { });

    start();
}

function start() {
    history.scrollRestoration = "auto";
    reset_controls();
    refresh();
};

window.onpopstate = async e => {
    await refresh();
    restore_scroll_position(e.state);
};

window.addEventListener("load", () => {
    if (is_desktop_mode()) return;
    start_mobile_mode();
});

// now returns the current time in seconds since the Unix epoch.
function now() {
    return Math.floor(new Date() / 1000);
}

// seconds_ago receives a timestamp and returns the number of seconds that have
// passed since the given timestamp. It returns a negative number if the given
// timestamp is in the future.
function seconds_ago(timestamp) {
    return now() - timestamp;
}

const seconds_in_a_minute = 60;
const seconds_in_an_hour = 60 * seconds_in_a_minute;
const seconds_in_a_day = 24 * seconds_in_an_hour;
const seconds_in_a_week = 7 * seconds_in_a_day;
const seconds_in_a_month = 30 * seconds_in_a_day;
const seconds_in_a_year = 365 * seconds_in_a_day;

const units = [
    [seconds_in_a_minute, "minute"],
    [seconds_in_an_hour, "hour"],
    [seconds_in_a_day, "day"],
    [seconds_in_a_week, "week"],
    [seconds_in_a_month, "month"],
    [seconds_in_a_year, "year"],
]

const rtf = new Intl.RelativeTimeFormat("en", {
    localeMatcher: "best fit",
    numeric: "auto",
    style: "long"
});

// relative_time_desc receives a timestamp and returns a human-readable string
// in English representing the time difference between the given timestamp and
// the current time (e.g., 3 hours ago).
function relative_time_desc(timestamp) {
    let diff = -seconds_ago(timestamp);

    let chosen_quantity = 1;
    let chosen_unit = "second";
    for (let i = 0; i < units.length; i++) {
        let [quantity, unit] = units[i];
        if (Math.abs(diff) < quantity) {
            break
        }
        chosen_quantity = quantity;
        chosen_unit = unit;
    }

    let adapted_diff = Math.round(diff / chosen_quantity);
    return rtf.format(adapted_diff, chosen_unit);
}
