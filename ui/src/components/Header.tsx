import { createDockerDesktopClient } from "@docker/extension-api-client";
import { QuestionAnswerOutlined } from "@mui/icons-material";
import { Grid, Link, Paper } from "@mui/material";
import Typography from "@mui/material/Typography/Typography";

const ddClient = createDockerDesktopClient();

const FEEDBACK_FORM_URL = "https://forms.gle/kYAwK34RUFXyAdfS7";
const LEARN_MORE_URL = "https://docs.docker.com/desktop/use-desktop/volumes/";

export const Header = () => (
  <>
    <Grid container gap={2} alignItems="center">
      <Typography variant="h3">Volumes Backup & Share</Typography>
      <Link
        href="#"
        onClick={() => {
          ddClient.host.openExternal(FEEDBACK_FORM_URL);
        }}
      >
        <Typography display="inline" variant="body2">
          Give Feedback
        </Typography>
        <QuestionAnswerOutlined
          fontSize="small"
          sx={{
            verticalAlign: "bottom",
            marginLeft: "0.25em",
          }}
        />
      </Link>
    </Grid>
    <Typography variant="body1" color="text.secondary" sx={{ mt: 2 }}>
      Backup, clone, restore, and share Docker volumes effortlessly.
    </Typography>
    <Paper
      sx={(theme) => ({
        mt: 2,
        background: theme.palette.docker.blue[100],
        color: theme.palette.docker.blue[700],
        border: "none",
      })}
    >
      <Typography variant="body1" sx={{ p: 2 }}>
        The functionality in this extension has been available in the Volumes
        tab of Docker Desktop in versions 4.29 and later.This extension will be
        deprecated and removed from the marketplace effective September 30,
        2024.{" "}
        <Link
          href="#"
          onClick={() => {
            ddClient.host.openExternal(LEARN_MORE_URL);
          }}
        >
          Learn more
        </Link>
      </Typography>
    </Paper>
  </>
);
