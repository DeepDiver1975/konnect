import React, { Component } from 'react';
import { connect } from 'react-redux';
import { withStyles } from 'material-ui/styles';
import PropTypes from 'prop-types';
import Button from 'material-ui/Button';
import List, { ListItem, ListItemText } from 'material-ui/List';
import Checkbox from 'material-ui/Checkbox';
import Tooltip from 'material-ui/Tooltip';
import { CircularProgress } from 'material-ui/Progress';
import green from 'material-ui/colors/green';
import Typography from 'material-ui/Typography';
import renderIf from 'render-if';

import { executeConsent, advanceLogonFlow } from '../actions/login-actions';
import { REQUEST_CONSENT_ALLOW } from '../actions/action-types';

const styles = theme => ({
  button: {
    margin: theme.spacing.unit
  },
  buttonProgress: {
    color: green[500],
    position: 'absolute',
    top: '50%',
    left: '50%',
    marginTop: -12,
    marginLeft: -12
  },
  buttonGroup: {
    textAlign: 'right'
  },
  subHeader: {
    marginBottom: theme.spacing.unit * 2
  },
  scopeList: {
    marginBottom: theme.spacing.unit * 2
  },
  wrapper: {
    marginTop: theme.spacing.unit * 5,
    position: 'relative',
    display: 'inline-block'
  },
  message: {
    marginTop: theme.spacing.unit * 2
  }
});

class Login extends Component {
  componentDidMount() {
    const { hello, history, client } = this.props;
    if ((!hello || !hello.state || !client) && history.action !== 'PUSH') {
      history.replace(`/identifier${history.location.search}${history.location.hash}`);
    }
  }

  render() {
    const { classes, loading, hello, errors, client } = this.props;
    return (
      <div>
        <Typography type="headline" component="h3">
          Hi {hello.displayName}
        </Typography>
        <Typography type="subheading" className={classes.subHeader}>
          {hello.username}
        </Typography>

        <Typography type="subheading" gutterBottom>
          <Tooltip placement="bottom" title={`Clicking "Allow" will redirect you to: ${client.uri}`}><ClientDisplayName client={client}/></Tooltip> wants to
        </Typography>
        <List dense disablePadding className={classes.scopeList}>
          <ListItem
            disableGutters
          ><Checkbox
              checked={true}
              disableRipple
              disabled
            />
            <ListItemText primary="Access your basic account information" />
          </ListItem>
        </List>

        <Typography type="subheading" gutterBottom>Allow <ClientDisplayName client={client}/> to do this?</Typography>
        <Typography color="secondary">By clicking Allow, you allow this app to use your information.</Typography>

        <form action="" onSubmit={(event) => this.logon(event)}>
          <div className={classes.buttonGroup}>
            <div className={classes.wrapper}>
              <Button
                color="primary"
                className={classes.button}
                disabled={!!loading}
                onClick={(event) => this.action(event, false)}
              >Cancel
              </Button>
              {(loading && loading !== REQUEST_CONSENT_ALLOW) && <CircularProgress size={24} className={classes.buttonProgress} />}
            </div>
            <div className={classes.wrapper}>
              <Button
                type="submit"
                raised
                color="primary"
                className={classes.button}
                disabled={!!loading}
                onClick={(event) => this.action(event, true)}
              >Allow</Button>
              {loading === REQUEST_CONSENT_ALLOW && <CircularProgress size={24} className={classes.buttonProgress} />}
            </div>
          </div>

          {renderIf(errors.http)(() => (
            <Typography type="body1" color="error" className={classes.message}>{errors.http.message}</Typography>
          ))}
        </form>
      </div>
    );
  }

  action(event, allow=false) {
    event.preventDefault();

    const { dispatch, history } = this.props;
    dispatch(executeConsent(allow)).then((response) => {
      if (response.success) {
        dispatch(advanceLogonFlow(response.success, history, true, {konnect: response.state}));
      }
    });
  }
}

Login.propTypes = {
  classes: PropTypes.object.isRequired,

  loading: PropTypes.string.isRequired,
  errors: PropTypes.object.isRequired,
  hello: PropTypes.object,
  client: PropTypes.object.isRequired,

  dispatch: PropTypes.func.isRequired,
  history: PropTypes.object.isRequired
};

const ClientDisplayName = ({ client, ...rest }) => (
  <span {...rest}>{client.display_name ? client.display_name : client.id}</span>
);

ClientDisplayName.propTypes = {
  client: PropTypes.object.isRequired
};

const mapStateToProps = (state) => {
  const { hello } = state.common;
  const { loading, errors } = state.login;

  return {
    loading: loading,
    errors,
    hello,
    client: hello.details.client || {}
  };
};

export default connect(mapStateToProps)(withStyles(styles)(Login));