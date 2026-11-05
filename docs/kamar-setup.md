# Connecting KAMAR to Athletics Day

For the school's KAMAR administrator. This lets KAMAR send the student list to Athletics Day automatically, so nobody has to type it in or import a CSV.

## What is shared

Only four things about each student:

| Item | KAMAR field |
|---|---|
| Preferred first name | `firstname` |
| Last name | `lastname` |
| Date of birth | `datebirth` |
| Group (tutor class, 3 letters like SSH) | `tutor` |

Nothing else is requested: no photos, timetables, attendance, pastoral notes, caregivers, addresses, ethnicity or staff data.
Students who have opted out in the Data Sharing Service screen are not sent by KAMAR.

The tutor class must be exactly 3 letters, and its first letter is the house (S Stuart, T Tudor, W Windsor, H Hanover). Students whose tutor class looks different are skipped. The organiser can add them by hand.

## Set it up in KAMAR

Setup > Server > Directory Services

1. Add a new service with these details (the organiser will give you the real values in person):
   - **Server address:** `https://<athletics server>/api/kamar` (the organiser fills this in)
   - **Username / Password:** supplied by the organiser
   - **Format:** **JSON** (XML is not supported)
   - **Updates:** both full and part updates are fine
2. Press **Check and Enable**.
3. Choose the data to send: **Student details** with the fields above. Do not tick extra fields.
4. Press **Run Now** to send everyone straight away. After that KAMAR updates the list nightly and when a student changes.

If the data boxes are greyed out after the check, tell the organiser. See "If something looks wrong" below.

## Privacy

- KAMAR requires a memorandum of understanding with the school covering what is collected, why, where it is stored, how long it is kept, and what happens in a breach. The school and organiser should complete this before enabling the service.
- Data is used only to organise the athletics day. Delete it once the event is over (Organiser > Students > Remove all students).
- If the school deletes the service in KAMAR, Athletics Day removes every student that came from KAMAR.

## If something looks wrong

| Symptom | What to check |
|---|---|
| Check fails | Address, username, password. Make sure the address is reachable from the KAMAR server and uses HTTPS |
| Service switches itself off | The format must be JSON. Re-run Check and Enable |
| Data boxes greyed out | Run Check and Enable again. The organiser can then try the next step in the notes below |
| Few or no students arrive | Press Run Now. Check the students' year levels are within the range (default Year 9 to 13) |
| Some students missing | Their tutor class is not 3 letters, or their date of birth is missing. See the server log |

Organiser notes: the server log shows what each message contained by field name and count, never by value. The Organiser > Students page shows when KAMAR last sent data. KAMAR's guidance is that the data boxes may only unlock once every option is sent explicitly, the student details field list is supplied, and the year-level range is included. This service sends all three.
