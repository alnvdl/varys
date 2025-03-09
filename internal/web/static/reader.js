const TOKEN_PREFIX = "token:";
const HOME_PATH = "/feeds/all";
// REFRESH_TIMEOUT is the amount of time in seconds after which cache entries
// should expire and the page should reload when focused.
const REFRESH_TIMEOUT = 60 * 60; // 1 hour

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

function time_diff_s(a, b) {
    return Math.round((a - b) / 1000);
}

function now() {
    return Math.floor(Date.now()/1000);
}

const CACHE_LIST = "list"
const CACHE_FEED = "feed"
const CACHE_ITEM = "item"

// FeedStore is a simple in-memory cache for feed data that reaches out to the
// backend when data is not available or has expired.
class FeedStore {
    constructor() {
        this.entries = {};
    }

    _put(key, data) {
        this.entries[key] = {
            ts: new Date(),
            data,
        };
    }

    _get(key) {
        let entry = this.entries[key];
        if (!entry) return entry;
        let now = new Date();
        let age = time_diff_s(now, entry.timestamp);
        if (age >= REFRESH_TIMEOUT) {
            delete this.entries[key];
            return undefined;
        }
        return entry.data;
    }

    _mark_as_read_items_matching(cond) {
        Object.values(this.entries).forEach(entry => {
            entry.data.items?.forEach(item => {
                if (cond(item)) {
                    item.read = 1;
                    let feed = this._get(item.feed_uid);
                    if (feed) feed.read_count += 1;
                }
            });
        });
    }

    async _fetch(uid, url, use_cache) {
        let cached = this._get(uid);
        if (use_cache && cached) {
            return [{status: 200}, cached];
        }
        let rsp = await fetch(url);
        let data = await rsp.json();
        if (rsp.status === 200) {
            this._put(uid, data);
        }
        return [rsp, data];
    }

    async mark_as_read(what, fuid, iuid) {
        if (what == CACHE_LIST) {
            what = CACHE_FEED;
            fuid = "all";
        }

        if (what == CACHE_FEED) {
            let before = this.entries[fuid]?.data?.last_updated || now();
            let rsp = await fetch(`/api/feeds/${fuid}/read`, {
                method: "POST",
                body: JSON.stringify({before}),
                headers: {"Content-Type": "application/json"},
            });
            if (rsp.status !== 200) {
                return false;
            }
            this._mark_as_read_items_matching(item => fuid === "all" || item.feed_uid === fuid)
        }

        if (what == CACHE_ITEM) {
            let rsp = await fetch(`/api/feeds/${fuid}/items/${iuid}/read`, {method: "POST"});
            if (rsp.status !== 200) {
                return false;
            }
            this._mark_as_read_items_matching(item => item.uid === iuid)
        }

        return true;
    }

    async fetch_feed(uid, use_cache) {
        return this._fetch(uid, `/api/feeds/${uid}`, use_cache);
    }

    async fetch_feeds() {
        // Due to multiple bugs (cache invalidation is hard...), caching was
        // disabled for the feed list.
        return this._fetch("feeds", `/api/feeds`, false);
    }
}

// Page state.
var previous_hash = window.location.hash;
var current_hash = window.location.hash;
var feed_cache = new FeedStore();
var last_refresh = new Date();

function error(err) {
    console.trace(err);
    let div = document.createElement("div");
    div.setAttribute("class", "error");
    div.appendChild(document.createTextNode(err));
    set_content(div);
    reset_controls({breadcrumbs: true});
}

async function show_feeds(use_cache) {
    let rsp, data;
    try {
        reset_controls({breadcrumbs: true, read_button: true});
        set_loading();
        [rsp, data] = await feed_cache.fetch_feeds(use_cache);
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
            reset_controls({breadcrumbs: true, read_button: true});
            let feed_list = document.createElement("ol");
            feed_list.setAttribute("class", "feed-list")
            data.forEach(feed => {
                let li = document.createElement("li");

                let a = document.createElement("a");
                a.appendChild(document.createTextNode(feed.name));
                link(a, `/feeds/${feed.uid}`);

                let unread = feed.item_count - feed.read_count;
                if (unread) {
                    let span = document.createElement("span");
                    span.setAttribute("class", "feed-unread-count")
                    span.appendChild(document.createTextNode(unread));
                    a.appendChild(span);
                } else {
                    li.setAttribute("class", "feed-read")
                }

                li.appendChild(a);
                feed_list.appendChild(li);
            })
            set_content(feed_list);
            break;
        default:
            error(`Unexpected response code: ${rsp.status}`);
            break;
    }
}

async function show_feed(uid, use_cache) {
    let rsp, data;
    try {
        reset_controls({breadcrumbs: true, read_button: true});
        set_loading();
        [rsp, data] = await feed_cache.fetch_feed(uid, use_cache);
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
            reset_controls({breadcrumbs: {uid: data.uid, name: data.name}, read_button: true});
            let item_list = document.createElement("div");
            item_list.classList = "item-list";
            data.items?.forEach(item => {
                item_list.appendChild(gen_item(item, {list_view: true}));
            });
            set_content(item_list);
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
        rsp = await fetch(`/api/feeds/${fuid}/items/${iuid}`);
        data = await rsp.json();
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
                breadcrumbs: {uid: data.feed, name: data.feed_name},
                open_button: data.url,
            });
            set_content(gen_item(data));
            window.scrollTo(0, 0);
            if (!data.read) {
                feed_cache.mark_as_read(CACHE_ITEM, data.feed_uid, data.uid);
            }
            break;
        default:
            error(`Unexpected response code: ${rsp.status}`);
            break;
    }
}

async function mark_all_as_read() {
    set_loading();
    let state = get_current_state();
    if (state.parts.length == 1 && state.parts[0] == "feeds") {
        await feed_cache.mark_as_read(CACHE_LIST);
    } else if (state.parts.length == 2 && state.parts[0] == "feeds") {
        await feed_cache.mark_as_read(CACHE_FEED, state.parts[1]);
    }
    refresh();
}

function gen_item(item, opts) {
    let list_view = (opts && opts.list_view) || false;

    let itemDiv = document.createElement("div");
    itemDiv.classList.add("item");

    let headerDiv = document.createElement("div");
    headerDiv.classList.add("item-header");
    itemDiv.appendChild(headerDiv);;

    let titleDiv = document.createElement("div");
    titleDiv.classList.add("item-title");
    headerDiv.appendChild(titleDiv);;

    let detailsDiv = document.createElement("div");
    detailsDiv.classList.add("item-details");
    headerDiv.appendChild(detailsDiv);;

    let contentDiv = document.createElement("div");
    contentDiv.classList.add("item-content");
    itemDiv.append(contentDiv);

    let title = document.createTextNode(item.title);
    titleDiv.appendChild(title);
    if (!list_view) {
        titleDiv.classList.add("item-title-bold");
    }

    let details = [];
    details.push(item.feed_name);

    if (item.authors) {
        if (item.authors.length > 32) {
            item.authors = item.authors.substr(0, 32) + "...";
        }
        details.push("by " + item.authors);
    }

    let ts = fromNow(new Date(parseInt(item.timestamp) * 1000));
    details.push(ts);

    let detailsNode = document.createTextNode(details.join(" · "));
    detailsDiv.appendChild(detailsNode);

    if (!list_view) {
        contentDiv.innerHTML = item.content;
        itemDiv.setAttribute("class", "item-full");
    } else {
        itemDiv.setAttribute("class", "item-summary");
        contentDiv.parentNode.removeChild(contentDiv);
    }

    if (list_view) {
        let a = document.createElement("a");
        link(a, `/feeds/${item.feed_uid}/items/${item.uid}`);
        a.setAttribute("class", "item-link");
        a.appendChild(itemDiv);
        itemDiv = a;
        if (item.read) {
            a.setAttribute("class", a.getAttribute("class") + " item-link-read");
        }
    }

    return itemDiv;
}

async function refresh(use_cache) {
    let state = get_current_state();
    if (state.login) {
        let rsp = await fetch(`/login`, {
            method: "POST",
            headers: {"Content-Type": "application/json"},
            body: JSON.stringify({token: state.token})
        });
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
    } else if (state.parts[0] === "feeds") {
        if (state.parts[1]) {
            await show_feed(state.parts[1], use_cache);
        } else {
            await show_feeds(use_cache);
        }
    } else {
        error("Please login.");
    }
}

let refresh_using_cache = () => {refresh(true)};

function set_content(...content) {
    let elem = document.querySelector("#content");
    while (elem.firstChild) {
        elem.removeChild(elem.lastChild);
    }
    content.forEach(child => elem.appendChild(child));
}

function set_loading() {
    let div = document.createElement("div");
    div.classList.add("loading");
    let spinner = document.createElement("div");
    spinner.classList.add("spinner");
    div.appendChild(spinner);
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
        elem.appendChild(div);
    } else {
        set_content(div);
    }
}

function link(a, url) {
    a.setAttribute("href", url);
    if (a.onclick) return;
    a.onclick = e => {
        history.pushState(null, "", a.href);
        refresh_using_cache();
        e.preventDefault();
    };
}

function reset_controls(config) {
    if (!config) {
        document.querySelector("#controls").classList.add("hidden");
        return;
    }

    document.querySelector("#controls").classList.remove("hidden");

    let breadcrumbs = document.querySelector(`#breadcrumbs`);
    breadcrumbs.classList.add("hidden");
    if (config.breadcrumbs) {
        breadcrumbs.classList.remove("hidden");

        let items = document.querySelector("#breadcrumb-items");
        items.textContent = "";

        let li = document.createElement("li");
        li.setAttribute("class", "breadcrumb-item");
        a = document.createElement("a");
        link(a, "/feeds");
        a.appendChild(document.createTextNode("Feeds"));
        li.appendChild(a);
        items.appendChild(li);

        if (typeof config.breadcrumbs === 'object' && config.breadcrumbs !== null) {
            li = document.createElement("li");
            li.setAttribute("class", "breadcrumb-item");
            a = document.createElement("a");
            link(a, `/feeds/${config.breadcrumbs.uid}`);
            let name = config.breadcrumbs.name;
            a.appendChild(document.createTextNode(`${name}`));
            li.appendChild(a);
            items.appendChild(li);
        }
    }

    let read_button = document.querySelector(`#read-button`);
    read_button.classList.add("hidden");
    if (config.read_button) {
        read_button.classList.remove("hidden");
    }

    let open_button = document.querySelector(`#open-button`);
    open_button.classList.add("hidden");
    if (config.open_button) {
        link(open_button, config.open_button);
        open_button.setAttribute("target", "_blank");
        open_button.setAttribute("rel", "noopener noreferrer");
        open_button.classList.remove("hidden");
    }
}

function start() {
    history.scrollRestoration = "auto";
    reset_controls();
    refresh();
};

window.onpopstate = refresh_using_cache;

window.onload = () => {
    let read_button = document.querySelector("#read-button");
    read_button.addEventListener("click", mark_all_as_read);
    read_button.addEventListener("touchstart", () => { });

    let open_button = document.querySelector("#open-button");
    open_button.addEventListener("touchstart", () => { });

    start();
};

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

function fromNow(datetime) {
    let now = new Date();
    let diff = time_diff_s(datetime, now);

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
