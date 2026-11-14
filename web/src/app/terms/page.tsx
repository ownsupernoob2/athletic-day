import type { Metadata } from "next";
import Link from "next/link";
import { Contact, LegalPage } from "@/components/LegalPage";
import { site } from "@/lib/site";

export const metadata: Metadata = { title: `Terms of service | ${site.name}` };

export default function TermsPage() {
  return (
    <LegalPage title="Terms of service">
      <p>
        These are the rules for using {site.name}, a website that helps {site.school} organise its athletics day. It is run by {site.operator} for the school. By signing in or using the website you agree to them. If you do not agree, please do not use it.
      </p>

      <h2>Who can use it</h2>
      <ul>
        <li>
          <strong>Anyone</strong> can look at the published schedule and download or print it.
        </li>
        <li>
          <strong>Staff</strong> use an account to choose events for the students in their group.
        </li>
        <li>
          <strong>The organiser</strong> manages students, events, accounts and the schedule.
        </li>
      </ul>
      <p>
        Accounts are made by the organiser, for staff of {site.school}. You cannot sign up for yourself.
      </p>

      <h2>Your account</h2>
      <ul>
        <li>Keep your password private and do not let anyone else use your account.</li>
        <li>Tell us straight away if you think someone else has used your account.</li>
        <li>The organiser can change, suspend or remove an account at any time, for example when someone leaves the school.</li>
      </ul>

      <h2>What you must and must not do</h2>
      <ul>
        <li>Use the website only to organise the athletics day.</li>
        <li>Only change entries for students in your own group, unless you are the organiser.</li>
        <li>Enter accurate information, and only for students who belong at the school.</li>
        <li>Keep student information inside the school. Do not copy it, share it or publish it elsewhere, apart from the published schedule and programme.</li>
        <li>Do not try to get into accounts or data you are not meant to see, break or slow down the website, or run automated tools against it.</li>
        <li>Do not use the website in a way that is unlawful or that harms students or staff.</li>
      </ul>
      <p>If you break these rules, your access can be removed.</p>

      <h2>Student information</h2>
      <p>
        How we handle personal information is explained in our <Link href="/privacy">privacy policy</Link>, which is part of these terms. {site.school} is responsible for deciding which students are included and for the accuracy of what the school provides.
      </p>

      <h2>The schedule</h2>
      <p>
        Heats, lanes and start times are set by the school&apos;s organisers and can change at any time, including on the day. Check with the organisers if you are unsure. Programmes you print show the schedule at the time you printed them.
      </p>

      <h2>Availability</h2>
      <p>
        The website is free and we try to keep it working, but we do not promise it will always be available or free of mistakes. We may change or stop it. While it is being tested, information may be reset or removed without warning, so keep your own copy of anything important (staff and the organiser can download CSV files).
      </p>

      <h2>Name and crest</h2>
      <p>
        The {site.school} name and crest belong to {site.school}. Please do not use them for anything else without the school&apos;s permission.
      </p>

      <h2>Our responsibility</h2>
      <p>
        We take reasonable care, but the website is provided &ldquo;as is&rdquo;. To the extent New Zealand law allows, we are not responsible for losses that come from using it or not being able to use it, for example a wrong time or a missed event. Nothing in these terms limits any right you have under New Zealand law that cannot be excluded.
      </p>

      <h2>Changes</h2>
      <p>We may change these terms. We will update the date at the top, and continuing to use the website after a change means you accept it.</p>

      <h2>Law</h2>
      <p>These terms are governed by New Zealand law, and New Zealand courts decide any dispute about them.</p>

      <h2>Contact</h2>
      <p>
        {site.operator}
        <br />
        <Contact />
      </p>
    </LegalPage>
  );
}
