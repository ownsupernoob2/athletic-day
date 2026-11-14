import type { Metadata } from "next";
import { Contact, LegalPage } from "@/components/LegalPage";
import { site } from "@/lib/site";

export const metadata: Metadata = { title: `Privacy policy | ${site.name}` };

export default function PrivacyPage() {
  return (
    <LegalPage title="Privacy policy">
      <p>
        This explains what personal information {site.name} uses, why, who can see it and how to ask about it. {site.name} is a website that helps {site.school} organise its athletics day: who is doing which events, the schedule, and the printed programme. It is run by {site.operator} for the school.
      </p>
      <p>
        The school decides whose information goes into the system and why. {site.operator} runs the website and looks after the information on the school&apos;s behalf. We follow the New Zealand Privacy Act 2020.
      </p>

      <h2>What information we hold</h2>
      <p>
        <strong>About students.</strong> For each student taking part:
      </p>
      <ul>
        <li>preferred first name and last name</li>
        <li>date of birth (used to work out their age group, for example Under 14)</li>
        <li>tutor group (for example SSH). The first letter of the group tells us the house.</li>
        <li>the events they have chosen, and their heat and start time once the schedule is made</li>
        <li>their KAMAR student number, if the details came from KAMAR (so we can keep them up to date)</li>
      </ul>
      <p>
        This comes from the school&apos;s student management system (KAMAR), a spreadsheet the organiser uploads, or a teacher adding a student who is missing from their group. We do <strong>not</strong> ask for photos, timetables, attendance, pastoral or health notes, addresses, caregiver details, ethnicity or anything about staff from KAMAR.
      </p>
      <p>
        <strong>About staff and organisers.</strong> Name, school email address, the group they look after, and a password. Passwords are stored scrambled (hashed), so nobody, including us, can read them. When you sign in we keep a sign-in session for up to 14 days.
      </p>
      <p>
        <strong>About how the website is used.</strong> The server keeps a short list of recent actions in its memory (for example who signed in, and which events were ticked for which student) to help us see what the website is doing and fix problems. It is cleared whenever the server restarts and is only shown on test versions of the site. Our hosting providers (below) can see technical details such as your IP address when you visit.
      </p>
      <p>
        We use one cookie, which keeps you signed in. It is needed for the site to work, so there is no cookie banner. We do not use advertising, analytics or tracking tools.
      </p>

      <h2>Why we use it</h2>
      <ul>
        <li>to let teachers choose events for the students in their group</li>
        <li>to let the organiser see who has finished, build the heats and times, and fix clashes</li>
        <li>to publish the schedule and print the programme booklet</li>
        <li>to export entries for the school&apos;s own records</li>
      </ul>
      <p>We do not use it for anything else, sell it, or use it for marketing.</p>

      <h2>Who can see it</h2>
      <ul>
        <li>
          <strong>Teachers</strong> see only the students in their own group.
        </li>
        <li>
          <strong>The organiser</strong> sees everyone, and manages the accounts.
        </li>
        <li>
          <strong>Anyone with the link</strong> can see the published schedule without signing in. It shows each student&apos;s first name, last name, tutor group, house, age group, events and start times. It does <strong>not</strong> show date of birth. The organiser can unpublish it at any time.
        </li>
        <li>
          <strong>The printed programme</strong> shows the same details as the published schedule.
        </li>
      </ul>

      <h2>Who else handles it</h2>
      <p>We do not share information with anyone else, except the services needed to run the website:</p>
      <ul>
        <li>
          <strong>KAMAR</strong>, the school&apos;s student management system, sends student details to us. We do not send anything back.
        </li>
        <li>
          <strong>Vercel</strong> hosts the website pages, and requests from your browser pass through it on the way to our server.
        </li>
        <li>
          <strong>Cloudflare</strong> provides the secure connection to our server.
        </li>
      </ul>
      <p>
        The student database is stored on a server we run in New Zealand. Because Vercel and Cloudflare are international companies, the information you send or receive while using the site can pass through computers outside New Zealand on its way.
      </p>

      <h2>How we protect it</h2>
      <ul>
        <li>connections are encrypted (HTTPS)</li>
        <li>passwords are hashed, and repeated wrong passwords are slowed down</li>
        <li>people only see what their role allows</li>
        <li>KAMAR can only send us data if it knows a password that the school holds</li>
      </ul>
      <p>
        No system is completely secure. If a privacy breach is likely to cause serious harm, we will tell the school, the people affected and the Privacy Commissioner, as the Privacy Act requires.
      </p>

      <h2>How long we keep it</h2>
      <p>
        We keep student information only for as long as it is needed for the athletics day. After the event the organiser removes all students (there is a button for this), or the school can ask us to. If the school switches off the KAMAR connection, we delete everything that came from KAMAR. Students who leave the school are removed when KAMAR tells us they have left. When we delete the information we also delete the backup copies we hold. Staff accounts are deleted when the organiser removes them.
      </p>

      <h2>Your rights</h2>
      <p>
        You can ask to see the information we hold about you or your child, and ask us to correct it. A parent or caregiver can ask on behalf of a student. You can also ask us to delete information. Email <Contact /> and we will reply within 20 working days. You can also ask {site.school}, who can change a student&apos;s details at the source in KAMAR.
      </p>
      <p>
        If you are unhappy with how we handled your information, please tell us first. You can also complain to the Office of the Privacy Commissioner at <a href="https://www.privacy.org.nz">privacy.org.nz</a>.
      </p>

      <h2>Children</h2>
      <p>
        Most people on this website are students under 18. Students do not create accounts and we never contact students. Their information comes from the school, and only the people the school gives accounts to can see more than the published schedule.
      </p>

      <h2>Changes</h2>
      <p>If we change this policy, we will update the date at the top. If a change is important, we will tell the school.</p>

      <h2>Contact</h2>
      <p>
        {site.operator}
        <br />
        <Contact />
      </p>
    </LegalPage>
  );
}
