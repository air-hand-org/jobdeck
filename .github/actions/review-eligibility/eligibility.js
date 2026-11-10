module.exports = async ({ github, context }) => {
  let number;
  if (context.eventName === 'issue_comment') {
    if (context.payload.action !== 'created' || !context.payload.issue?.pull_request ||
        context.payload.comment?.body !== '/review') return null;
    const { data: access } = await github.rest.repos.getCollaboratorPermissionLevel({
      ...context.repo, username: context.payload.comment.user.login,
    });
    if (!['write', 'maintain', 'admin'].includes(access.permission) && !access.user?.permissions?.push) return null;
    number = context.payload.issue.number;
  } else if (context.eventName === 'pull_request') {
    const eventPr = context.payload.pull_request;
    if (eventPr?.head.repo?.full_name !== `${context.repo.owner}/${context.repo.repo}`) return null;
    number = eventPr.number;
  } else return null;
  const { data: pr } = await github.rest.pulls.get({ ...context.repo, pull_number: number });
  if (context.eventName === 'pull_request' && (
    pr.state !== 'open' || pr.merged || pr.draft || pr.auto_merge !== null ||
    pr.head.repo?.full_name !== `${context.repo.owner}/${context.repo.repo}` ||
    pr.head.sha !== context.payload.pull_request.head.sha
  )) return null;
  return true;
};
