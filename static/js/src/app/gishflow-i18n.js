/*
    gishflow-i18n.js
    Lightweight EN/TH translation for the GishFlow admin UI. It translates the
    static UI chrome (top nav, corner menu, page headers, and common
    buttons/labels) by exact-string dictionary lookup. Dynamic or user-entered
    data (campaign names, recipients, table rows loaded over AJAX) stays in
    English. The choice persists in localStorage; switching reloads the page so
    English is shown natively and Thai is applied on top.
*/
(function () {
    var TH = {
        // Top navigation
        "Dashboard": "แดชบอร์ด",
        "Campaigns": "แคมเปญ",
        "Risk": "ความเสี่ยง",
        "Users & Groups": "ผู้ใช้และกลุ่ม",
        "Email Templates": "เทมเพลตอีเมล",
        "Landing Pages": "หน้าเว็บปลอม",
        "Training Pages": "หน้าอบรม",
        "Sending Profiles": "โปรไฟล์การส่ง",
        // Corner menu
        "QR Generator": "เครื่องสร้าง QR",
        "Settings": "ตั้งค่า",
        "Users": "ผู้ใช้",
        "Webhooks": "เว็บฮุก",
        // Dashboard
        "Campaign timeline": "ไทม์ไลน์แคมเปญ",
        "events across all campaigns": "เหตุการณ์จากทุกแคมเปญ",
        "Recent campaigns": "แคมเปญล่าสุด",
        "All channels": "ทุกช่องทาง",
        "Sent": "ส่งแล้ว",
        "Opened": "เปิดอ่าน",
        "Clicked": "คลิก",
        "Submitted data": "กรอกข้อมูล",
        "Reported": "รายงาน",
        "QR scans": "สแกน QR",
        "View all": "ดูทั้งหมด",
        // Common actions ("New X" button labels, then modal titles)
        "New Campaign": "สร้างแคมเปญ",
        "New Group": "สร้างกลุ่ม",
        "New Template": "สร้างเทมเพลต",
        "New Page": "สร้างหน้าเว็บปลอม",
        "New Profile": "สร้างโปรไฟล์การส่ง",
        "New User": "สร้างผู้ใช้",
        "New Webhook": "สร้างเว็บฮุก",
        "New Training Page": "สร้างหน้าอบรม",
        "New Landing Page": "สร้างหน้าเว็บปลอม",
        "New Sending Profile": "สร้างโปรไฟล์การส่ง",
        "Export PDF": "ส่งออก PDF",
        "Save": "บันทึก",
        "Cancel": "ยกเลิก",
        // Risk page
        "Average risk": "ความเสี่ยงเฉลี่ย",
        "High-risk people": "คนเสี่ยงสูง",
        "People scored": "จำนวนคนที่ประเมิน",
        "need attention": "ต้องดูแล",
        "across all campaigns": "จากทุกแคมเปญ",
        "Distribution": "การกระจาย",
        "Recipients": "ผู้รับ",
        "Groups": "กลุ่ม",
        "Risk over time": "ความเสี่ยงตามเวลา",
        "score per campaign": "คะแนนต่อแคมเปญ",
        "Low": "ต่ำ",
        "Medium": "กลาง",
        "High": "สูง",
        // Risk page decay note (whitespace-normalized so the multi-line source matches)
        "Scores are recency-weighted — an action's weight halves every 30 days, so risk decays over time when a recipient stops falling for simulations, and climbs again if they slip up.":
            "คะแนนถ่วงน้ำหนักตามความใหม่ — น้ำหนักของแต่ละการกระทำจะลดลงครึ่งหนึ่งทุก 30 วัน ความเสี่ยงจึงค่อย ๆ ลดลงเมื่อผู้รับไม่ตกเป็นเหยื่ออีก และเพิ่มขึ้นอีกครั้งเมื่อพลาดซ้ำ",
        // Misc
        "Log out": "ออกจากระบบ"
    };

    function getLang() {
        try { return localStorage.getItem('gf-lang') || 'en'; } catch (e) { return 'en'; }
    }

    // gfToggleLang flips the language and reloads so English renders natively.
    window.gfToggleLang = function () {
        var next = getLang() === 'th' ? 'en' : 'th';
        try { localStorage.setItem('gf-lang', next); } catch (e) {}
        location.reload();
    };

    // The button shows the language you would switch TO.
    function setIcon() {
        var el = document.getElementById('gf-lang-ico');
        if (el) { el.textContent = getLang() === 'th' ? 'EN' : 'ไทย'; }
    }

    // translate walks every text node under root and replaces exact dictionary
    // matches, preserving surrounding whitespace.
    function translate(root) {
        var walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, null, false);
        var nodes = [], n;
        while ((n = walker.nextNode())) { nodes.push(n); }
        nodes.forEach(function (node) {
            var raw = node.nodeValue, key = raw.trim();
            if (!key) { return; }
            // Try an exact match, then a whitespace-normalized one so multi-line
            // paragraphs (e.g. the Risk note) still match a single-line key.
            var val = TH.hasOwnProperty(key) ? TH[key] : TH[key.replace(/\s+/g, ' ')];
            if (val) {
                node.nodeValue = raw.replace(key, val);
            }
        });
    }

    document.addEventListener('DOMContentLoaded', function () {
        setIcon();
        if (getLang() === 'th') {
            translate(document.body);
        }
    });
})();
